// SPDX-License-Identifier: PolyForm-Noncommercial-1.0.0
// Copyright © 2026 Nik m (@mazurovn). All rights reserved.
//
// client.ts — a thin, typed wrapper around the `mazzy-vpn` Go CLI. The pi
// extension never re-implements VPN logic; it drives the audited CLI (2.3.0+)
// through its stable --json contracts and privileged subcommands. Read-only
// commands (status/best/verify/test) need no root; mutating ones (up/daemon/
// disconnect/recover) elevate via the CLI's own sudo/pkexec path.

import type { ExtensionAPI } from "@earendil-works/pi-coding-agent";

/** The VPN connection state as reported by `mazzy-vpn status --json`. */
export type VpnState = "protected" | "link-up" | "down" | "unknown";

export interface VpnStatus {
	state: VpnState;
	interface: string;
	egress: string;
	profile: string;
}

export interface BestZone {
	name: string;
	endpoint: string;
	reachable: boolean;
	latency_ms: number;
	icmp_alive: boolean;
}

export interface ProfileAudit {
	name: string;
	protocol: string;
	country?: string;
	endpoint?: string;
	parses: boolean;
	valid: boolean;
	connectable: boolean;
	endpoint_dns: boolean;
	problems?: string[];
	advice?: string;
}

export interface ExecResult {
	stdout: string;
	stderr: string;
	/** Process exit code; null only when the command could not be spawned. */
	code: number | null;
	killed?: boolean;
}

/** Resolve the CLI binary once; honor MAZZY_VPN_BIN for non-standard installs. */
export function cliBinary(): string {
	return process.env.MAZZY_VPN_BIN?.trim() || "mazzy-vpn";
}

/**
 * MazzyClient wraps the CLI. All methods are best-effort and never throw on a
 * non-zero exit: they return structured results so the extension can render a
 * clear state instead of crashing the session.
 */
export class MazzyClient {
	constructor(private readonly pi: ExtensionAPI) {}

	private async run(args: string[], timeoutMs = 60_000, signal?: AbortSignal): Promise<ExecResult> {
		try {
			const r = await this.pi.exec(cliBinary(), args, { timeout: timeoutMs, signal });
			return { stdout: r.stdout ?? "", stderr: r.stderr ?? "", code: r.code, killed: r.killed };
		} catch (err) {
			return { stdout: "", stderr: err instanceof Error ? err.message : String(err), code: null };
		}
	}

	/** version returns the CLI version string, or "" when the CLI is missing. */
	async version(): Promise<string> {
		const r = await this.run(["version"], 5_000);
		if (r.code !== 0) return "";
		// Output form: "mazzy-vpn 2.3.0"
		return r.stdout.trim().split(/\s+/).pop() ?? "";
	}

	/** installed reports whether the CLI is on PATH (or MAZZY_VPN_BIN is valid). */
	async installed(): Promise<boolean> {
		return (await this.version()) !== "";
	}

	/** status reads the live connection state (read-only, no sudo). */
	async status(signal?: AbortSignal): Promise<VpnStatus> {
		const r = await this.run(["status", "--json"], 10_000, signal);
		if (r.code !== 0 || !r.stdout.trim()) {
			return { state: "unknown", interface: "", egress: "", profile: "" };
		}
		try {
			const j = JSON.parse(r.stdout) as Partial<VpnStatus>;
			const state = (j.state as VpnState) ?? "unknown";
			return {
				state: ["protected", "link-up", "down"].includes(state) ? state : "unknown",
				interface: j.interface ?? "",
				egress: j.egress ?? "",
				profile: j.profile ?? "",
			};
		} catch {
			return { state: "unknown", interface: "", egress: "", profile: "" };
		}
	}

	/** best ranks servers and returns the best live zone (read-only, no sudo). */
	async best(signal?: AbortSignal): Promise<BestZone | undefined> {
		const r = await this.run(["best", "--json"], 40_000, signal);
		if (r.code !== 0 || !r.stdout.trim()) return undefined;
		try {
			return JSON.parse(r.stdout) as BestZone;
		} catch {
			return undefined;
		}
	}

	/** verify audits every managed config (read-only, no sudo). */
	async verify(deepDns: boolean, signal?: AbortSignal): Promise<ProfileAudit[]> {
		const args = ["verify", "--json"];
		if (!deepDns) args.push("--no-dns");
		const r = await this.run(args, 60_000, signal);
		if (!r.stdout.trim()) return [];
		try {
			return JSON.parse(r.stdout) as ProfileAudit[];
		} catch {
			return [];
		}
	}

	/**
	 * connectBest starts the self-healing daemon on the best live zone in the
	 * background (survives the pi session). Elevates via the CLI's sudo/pkexec.
	 */
	async connectBest(signal?: AbortSignal): Promise<ExecResult> {
		return this.run(["daemon", "--best", "--background"], 120_000, signal);
	}

	/** connectZone starts the background daemon pinned to a named zone. */
	async connectZone(zone: string, signal?: AbortSignal): Promise<ExecResult> {
		return this.run(["daemon", zone, "--background"], 120_000, signal);
	}

	/** disconnect brings the tunnel down (records a durable down-intent). */
	async disconnect(signal?: AbortSignal): Promise<ExecResult> {
		return this.run(["disconnect"], 30_000, signal);
	}

	/** stop terminates a running background daemon. */
	async stop(signal?: AbortSignal): Promise<ExecResult> {
		return this.run(["stop"], 20_000, signal);
	}

	/** recover force-cleans all tunnels/guards (the panic button). */
	async recover(signal?: AbortSignal): Promise<ExecResult> {
		return this.run(["recover"], 30_000, signal);
	}

	/** doctor returns the host + catalog diagnostics as plain text. */
	async doctor(signal?: AbortSignal): Promise<string> {
		const r = await this.run(["doctor"], 20_000, signal);
		return (r.stdout || r.stderr || "").trim();
	}
}

/** A compact, human badge for a VPN state. */
export function stateBadge(s: VpnState): { glyph: string; label: string; tone: "success" | "warning" | "error" | "info" } {
	switch (s) {
		case "protected":
			return { glyph: "●", label: "PROTECTED", tone: "success" };
		case "link-up":
			return { glyph: "▲", label: "LINK-UP (no egress)", tone: "warning" };
		case "down":
			return { glyph: "○", label: "DISCONNECTED", tone: "error" };
		default:
			return { glyph: "?", label: "UNKNOWN", tone: "info" };
	}
}
