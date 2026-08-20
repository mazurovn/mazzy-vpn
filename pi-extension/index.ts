// SPDX-License-Identifier: PolyForm-Noncommercial-1.0.0
// Copyright © 2026 Nik m (@mazurovn). All rights reserved.
//
// Mazzy VPN — pi extension.
//
// Connect to and manage Mazzy VPN from inside pi. It drives the audited
// `mazzy-vpn` Go CLI (never re-implementing VPN logic) to:
//   - auto-connect to the best zone when the session starts unprotected,
//   - keep a live status widget/footer in the TUI,
//   - monitor the tunnel and auto-reconnect on egress loss ("if there is no
//     connection"),
//   - expose tools the model can call to connect / switch / disconnect /
//     recover / verify configs ("block or bypass"),
//   - offer /vpn commands and a Ctrl-based shortcut.
//
// Config (env, all optional):
//   MAZZY_VPN_BIN            path to the CLI (default: mazzy-vpn on PATH)
//   MAZZY_VPN_AUTOCONNECT    "1" (default) to auto-connect on session start
//   MAZZY_VPN_MONITOR        "1" (default) to run the background monitor
//   MAZZY_VPN_MONITOR_SECS   monitor interval seconds (default 30, min 10)
//   MAZZY_VPN_AUTORECONNECT  "1" (default) to auto-reconnect on egress loss

import type { ExtensionAPI, ExtensionContext } from "@earendil-works/pi-coding-agent";
import { Type } from "typebox";
import { StringEnum } from "@earendil-works/pi-ai";
import { MazzyClient, stateBadge, type VpnState, type VpnStatus } from "./client.ts";

const WIDGET_KEY = "mazzy-vpn";
const STATUS_KEY = "mazzy-vpn";

function envBool(name: string, dflt: boolean): boolean {
	const v = process.env[name]?.trim();
	if (v === undefined || v === "") return dflt;
	return v === "1" || v.toLowerCase() === "true" || v.toLowerCase() === "yes";
}

function monitorSecs(): number {
	const n = Number.parseInt(process.env.MAZZY_VPN_MONITOR_SECS ?? "", 10);
	if (!Number.isFinite(n) || n < 10) return 30;
	return n;
}

export default function mazzyVpnExtension(pi: ExtensionAPI) {
	const client = new MazzyClient(pi);

	// Session-scoped runtime state (never started from the factory itself).
	let timer: ReturnType<typeof setInterval> | undefined;
	let lastStatus: VpnStatus | undefined;
	let cliOk = false;
	let missingWarned = false;
	// Track consecutive unprotected observations so a brief blip does not trigger
	// a reconnect storm; mirror the CLI daemon's own patience.
	let unprotectedStreak = 0;
	let reconnecting = false;

	const autoConnect = envBool("MAZZY_VPN_AUTOCONNECT", true);
	const monitorOn = envBool("MAZZY_VPN_MONITOR", true);
	const autoReconnect = envBool("MAZZY_VPN_AUTORECONNECT", true);

	// ---- UI rendering -------------------------------------------------------

	function renderWidget(ctx: ExtensionContext, s: VpnStatus | undefined, note?: string): void {
		if (!ctx.hasUI) return;
		const theme = ctx.ui.theme;
		if (!s) {
			ctx.ui.setStatus(STATUS_KEY, theme.fg("dim", "Mazzy VPN: …"));
			return;
		}
		const b = stateBadge(s.state);
		// Map badge tone to a real theme color token.
		const tone: "success" | "warning" | "error" | "dim" =
			b.tone === "success" ? "success" : b.tone === "warning" ? "warning" : b.tone === "error" ? "error" : "dim";
		const badge = theme.fg(tone, `${b.glyph} ${b.label}`);
		// Footer: compact one-liner.
		let footer = `Mazzy VPN ${badge}`;
		if (s.state === "protected" && s.egress) footer += theme.fg("dim", ` · ${s.egress}`);
		ctx.ui.setStatus(STATUS_KEY, footer);

		// Widget above the editor: a couple of richer lines.
		const lines: string[] = [];
		const title = theme.fg("accent", theme.bold("Mazzy VPN"));
		lines.push(`${title}  ${badge}`);
		if (s.state === "protected") {
			lines.push(theme.fg("dim", `  iface ${s.interface || "?"} · egress ${s.egress || "?"}${s.profile ? ` · ${s.profile}` : ""}`));
		} else if (s.state === "link-up") {
			lines.push(theme.fg("dim", `  iface ${s.interface || "?"} · establishing / no egress yet`));
		} else if (s.state === "down") {
			lines.push(theme.fg("dim", `  not protected — /vpn connect or press the shortcut`));
		} else {
			lines.push(theme.fg("dim", `  status unknown — is mazzy-vpn installed?`));
		}
		if (note) lines.push(theme.fg("dim", `  ${note}`));
		ctx.ui.setWidget(WIDGET_KEY, lines);
	}

	async function refresh(ctx: ExtensionContext, note?: string): Promise<VpnStatus> {
		const s = await client.status(ctx.signal);
		lastStatus = s;
		renderWidget(ctx, s, note);
		return s;
	}

	// ---- Core actions -------------------------------------------------------

	async function ensureCli(ctx: ExtensionContext): Promise<boolean> {
		if (cliOk) return true;
		cliOk = await client.installed();
		if (!cliOk && !missingWarned) {
			missingWarned = true;
			if (ctx.hasUI) {
				ctx.ui.notify(
					"Mazzy VPN CLI not found. Install `mazzy-vpn` (or set MAZZY_VPN_BIN). VPN features are disabled.",
					"warning",
				);
				ctx.ui.setStatus(STATUS_KEY, ctx.ui.theme.fg("dim", "Mazzy VPN: CLI missing"));
			}
		}
		return cliOk;
	}

	async function connectBest(ctx: ExtensionContext): Promise<string> {
		if (!(await ensureCli(ctx))) return "Mazzy VPN CLI is not installed.";
		if (ctx.hasUI) ctx.ui.notify("Mazzy VPN: connecting to the best zone…", "info");
		renderWidget(ctx, lastStatus, "connecting to best zone…");
		const r = await client.connectBest(ctx.signal);
		// Give the daemon a moment to bring the tunnel up before sampling.
		await sleep(2500);
		const s = await refresh(ctx);
		if (s.state === "protected") return `Connected — protected via ${s.egress || s.interface}.`;
		if (r.code !== 0 && r.stderr.trim()) return `Connect requested but not confirmed yet (${firstLine(r.stderr)}).`;
		return "Connect requested; the tunnel is still establishing.";
	}

	async function connectZone(ctx: ExtensionContext, zone: string): Promise<string> {
		if (!(await ensureCli(ctx))) return "Mazzy VPN CLI is not installed.";
		if (ctx.hasUI) ctx.ui.notify(`Mazzy VPN: connecting to ${zone}…`, "info");
		renderWidget(ctx, lastStatus, `connecting to ${zone}…`);
		await client.connectZone(zone, ctx.signal);
		await sleep(2500);
		const s = await refresh(ctx);
		return s.state === "protected"
			? `Connected to ${zone} — protected via ${s.egress || s.interface}.`
			: `Requested ${zone}; still establishing.`;
	}

	async function disconnect(ctx: ExtensionContext): Promise<string> {
		if (!(await ensureCli(ctx))) return "Mazzy VPN CLI is not installed.";
		await client.disconnect(ctx.signal);
		await client.stop(ctx.signal); // also stop a background daemon if present
		await sleep(1200);
		const s = await refresh(ctx);
		return s.state === "down" || s.state === "unknown" ? "Disconnected." : "Disconnect requested.";
	}

	async function recover(ctx: ExtensionContext): Promise<string> {
		if (!(await ensureCli(ctx))) return "Mazzy VPN CLI is not installed.";
		if (ctx.hasUI) ctx.ui.notify("Mazzy VPN: recovering to plain uplink…", "warning");
		await client.recover(ctx.signal);
		await sleep(1200);
		await refresh(ctx);
		return "Recovered — all tunnels/guards cleared (plain uplink).";
	}

	// ---- Background monitor -------------------------------------------------

	function startMonitor(ctx: ExtensionContext): void {
		if (!monitorOn || timer) return;
		const interval = monitorSecs() * 1000;
		timer = setInterval(() => {
			void tick(ctx);
		}, interval);
		// Do not keep the process alive solely for the timer.
		if (typeof timer === "object" && "unref" in timer) (timer as { unref: () => void }).unref();
	}

	function stopMonitor(): void {
		if (timer) {
			clearInterval(timer);
			timer = undefined;
		}
	}

	async function tick(ctx: ExtensionContext): Promise<void> {
		if (!cliOk || reconnecting) return;
		const s = await refresh(ctx);
		if (s.state === "protected") {
			unprotectedStreak = 0;
			return;
		}
		// Not protected. Count consecutive misses; the CLI daemon already does
		// its own fast reconnect, so we only step in if it has stayed down.
		unprotectedStreak++;
		if (!autoReconnect || unprotectedStreak < 2) return;
		reconnecting = true;
		try {
			if (ctx.hasUI) ctx.ui.notify("Mazzy VPN: egress lost — reconnecting to the best zone…", "warning");
			await client.connectBest(ctx.signal);
			await sleep(2500);
			const after = await refresh(ctx);
			if (after.state === "protected") {
				unprotectedStreak = 0;
				if (ctx.hasUI) ctx.ui.notify(`Mazzy VPN: reconnected via ${after.egress || after.interface}.`, "info");
			}
		} finally {
			reconnecting = false;
		}
	}

	// ---- Lifecycle ----------------------------------------------------------

	pi.on("session_start", async (_event, ctx) => {
		// Defer all resource startup to here (never the factory).
		const ok = await ensureCli(ctx);
		if (!ok) return;
		const s = await refresh(ctx);

		// One-shot print/json runs (`-p`) have no user to protect and no place to
		// render a widget: report status only, never auto-connect or start a timer.
		if (!ctx.hasUI) return;

		const version = await client.version();
		ctx.ui.notify(`Mazzy VPN ${version} ready — ${stateBadge(s.state).label}.`, "info");

		// Auto-connect if unprotected and enabled ("connect automatically").
		if (autoConnect && s.state !== "protected" && s.state !== "link-up") {
			await connectBest(ctx);
		}
		startMonitor(ctx);
	});

	pi.on("session_shutdown", async (_event, _ctx) => {
		// Clean up the session-scoped timer. We intentionally do NOT tear down the
		// VPN: a background daemon should survive the pi session so the user stays
		// protected after they exit.
		stopMonitor();
	});

	// ---- Commands -----------------------------------------------------------

	const vpnCommand = {
		description: "Mazzy VPN: status | connect [zone] | disconnect | recover | verify | doctor",
		getArgumentCompletions: (prefix: string) => {
			const subs = ["status", "connect", "disconnect", "recover", "verify", "doctor", "best"];
			return subs
				.filter((s) => s.startsWith(prefix.trim()))
				.map((s) => ({ value: s, label: s }));
		},
		handler: async (args, ctx) => {
			const [sub, ...rest] = args.trim().split(/\s+/).filter(Boolean);
			switch (sub) {
				case undefined:
				case "status": {
					const s = await refresh(ctx);
					const b = stateBadge(s.state);
					ctx.ui.notify(
						`Mazzy VPN: ${b.glyph} ${b.label}` +
							(s.state === "protected" ? ` · ${s.interface} · ${s.egress}` : ""),
						s.state === "protected" ? "info" : "warning",
					);
					return;
				}
				case "connect": {
					const zone = rest.join(" ").trim();
					const msg = zone ? await connectZone(ctx, zone) : await connectBest(ctx);
					ctx.ui.notify(msg, "info");
					return;
				}
				case "disconnect":
					ctx.ui.notify(await disconnect(ctx), "info");
					return;
				case "recover":
					ctx.ui.notify(await recover(ctx), "warning");
					return;
				case "best": {
					if (!(await ensureCli(ctx))) return;
					const best = await client.best(ctx.signal);
					ctx.ui.notify(
						best ? `Best zone: ${best.name} (${best.latency_ms} ms${best.icmp_alive ? ", alive" : ""})` : "No reachable zone found.",
						best ? "info" : "warning",
					);
					return;
				}
				case "verify": {
					if (!(await ensureCli(ctx))) return;
					const audits = await client.verify(false, ctx.signal);
					const broken = audits.filter((a) => a.protocol !== "openvpn" && (!a.parses || !a.valid)).length;
					const ovpn = audits.filter((a) => a.protocol === "openvpn").length;
					const connectable = audits.filter((a) => a.connectable).length;
					ctx.ui.notify(
						`Configs: ${audits.length} total · ${connectable} connectable · ${ovpn} OpenVPN-only · ${broken} broken`,
						broken > 0 ? "warning" : "info",
					);
					return;
				}
				case "doctor": {
					if (!(await ensureCli(ctx))) return;
					const text = await client.doctor(ctx.signal);
					ctx.ui.notify(text || "doctor produced no output", "info");
					return;
				}
				default:
					ctx.ui.notify(`Unknown /mazzy-vpn subcommand: ${sub}`, "error");
			}
		},
	} as const;
	// Primary command is /mazzy-vpn; /vpn stays as a convenient alias.
	pi.registerCommand("mazzy-vpn", vpnCommand);
	pi.registerCommand("vpn", vpnCommand);

	// Quick toggle shortcut: connect when down, disconnect when protected.
	pi.registerShortcut("ctrl+alt+v", {
		description: "Mazzy VPN: toggle connect/disconnect",
		handler: async (ctx) => {
			if (!(await ensureCli(ctx))) return;
			const s = lastStatus ?? (await refresh(ctx));
			const msg = s.state === "protected" ? await disconnect(ctx) : await connectBest(ctx);
			ctx.ui.notify(msg, "info");
		},
	});

	// ---- Tools (LLM-callable) ----------------------------------------------

	pi.registerTool({
		name: "vpn_status",
		label: "Mazzy VPN Status",
		description: "Report the current Mazzy VPN connection state (protected/link-up/down), interface and egress IP. Read-only, no privileges required.",
		promptSnippet: "Check whether the Mazzy VPN tunnel is up and what the egress IP is",
		promptGuidelines: ["Use vpn_status before assuming network requests are or are not routed through the VPN."],
		parameters: Type.Object({}),
		async execute(_id, _params, signal, _onUpdate, ctx) {
			const s = await client.status(signal);
			renderWidget(ctx, s);
			return {
				content: [{ type: "text", text: `VPN ${s.state}${s.state === "protected" ? ` · ${s.interface} · egress ${s.egress}` : ""}` }],
				details: s,
			};
		},
	});

	pi.registerTool({
		name: "vpn_connect",
		label: "Mazzy VPN Connect",
		description:
			"Connect Mazzy VPN. With no zone, auto-picks the best live zone and starts the self-healing background daemon (auto-reconnect + failover). Optionally pin a named zone. Use this when there is no VPN connection or the user needs to bypass a block by routing through the VPN. Elevates via sudo/pkexec as the CLI requires.",
		promptSnippet: "Connect the VPN (best zone by default, or a named zone) to route traffic or bypass a block",
		promptGuidelines: [
			"Use vpn_connect when a request is blocked/geo-restricted and routing through the VPN could bypass it.",
			"Use vpn_connect with no zone to let Mazzy pick the fastest live server.",
		],
		parameters: Type.Object({
			zone: Type.Optional(Type.String({ description: "Managed zone name to connect to. Omit to auto-pick the best live zone." })),
		}),
		async execute(_id, params, signal, _onUpdate, ctx) {
			const zone = params.zone?.trim();
			const msg = zone ? await connectZone(ctx, zone) : await connectBest(ctx);
			return { content: [{ type: "text", text: msg }], details: { zone: zone ?? "(best)", status: lastStatus } };
		},
	});

	pi.registerTool({
		name: "vpn_disconnect",
		label: "Mazzy VPN Disconnect",
		description: "Disconnect Mazzy VPN and stop the background daemon, returning to the plain uplink. Elevates via sudo/pkexec.",
		promptSnippet: "Bring the VPN tunnel down and return to the normal connection",
		parameters: Type.Object({
			mode: Type.Optional(StringEnum(["graceful", "recover"] as const)),
		}),
		async execute(_id, params, signal, _onUpdate, ctx) {
			const msg = params.mode === "recover" ? await recover(ctx) : await disconnect(ctx);
			return { content: [{ type: "text", text: msg }], details: { status: lastStatus } };
		},
	});

	pi.registerTool({
		name: "vpn_verify_configs",
		label: "Mazzy VPN Verify Configs",
		description: "Audit every managed VPN profile: parses, valid, connectable, and endpoint DNS resolvability. Read-only. Use to diagnose why a zone will not connect.",
		promptSnippet: "Audit VPN profiles for validity and connectability",
		parameters: Type.Object({
			deepDns: Type.Optional(Type.Boolean({ description: "Also resolve each endpoint host in DNS (slower, more thorough)." })),
		}),
		async execute(_id, params, signal) {
			const audits = await client.verify(Boolean(params.deepDns), signal);
			// OpenVPN is catalogued-but-not-connectable (a known WARN), not "broken".
			// Only count real parse/validation failures on WG/AWG profiles as broken.
			const broken = audits.filter((a) => a.protocol !== "openvpn" && (!a.parses || !a.valid));
			const connectable = audits.filter((a) => a.connectable).length;
			const ovpn = audits.filter((a) => a.protocol === "openvpn").length;
			const summary = `Configs: ${audits.length} total · ${connectable} connectable · ${ovpn} OpenVPN-only · ${broken.length} broken`;
			const brokenList = broken.slice(0, 10).map((a) => `  ✖ ${a.name}: ${(a.problems ?? []).join("; ") || a.advice || "invalid"}`);
			return {
				content: [{ type: "text", text: [summary, ...brokenList].join("\n") }],
				details: { total: audits.length, connectable, openvpnOnly: ovpn, broken: broken.length, audits },
			};
		},
	});

	pi.registerTool({
		name: "vpn_best_zone",
		label: "Mazzy VPN Best Zone",
		description: "Rank managed servers and return the single best live zone (name, latency, liveness). Read-only.",
		promptSnippet: "Find the fastest live VPN zone without connecting",
		parameters: Type.Object({}),
		async execute(_id, _params, signal) {
			const best = await client.best(signal);
			return {
				content: [{ type: "text", text: best ? `Best: ${best.name} (${best.latency_ms} ms${best.icmp_alive ? ", alive" : ""})` : "No reachable zone found." }],
				details: best ?? {},
			};
		},
	});
}

// ---- small helpers ---------------------------------------------------------

function sleep(ms: number): Promise<void> {
	return new Promise((resolve) => setTimeout(resolve, ms));
}

function firstLine(s: string): string {
	return s.split("\n")[0]?.trim() ?? s.trim();
}
