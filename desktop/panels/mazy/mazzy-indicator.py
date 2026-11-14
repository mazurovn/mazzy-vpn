#!/usr/bin/env python3
# SPDX-License-Identifier: AGPL-3.0-or-later
# Copyright © 2026 Nik m (@mazurovn). All rights reserved.
#
# Mazy OS has no panel hook in this repo. Linux desktops that follow
# docs/DESKTOP.en.md show StatusNotifier/AppIndicator items in the top bar.
# This process registers that item. The label and menu are mazzy-panel.

import subprocess
import sys

try:
    import gi
    gi.require_version("AyatanaAppIndicator3", "0.1")
    gi.require_version("Gtk", "3.0")
    from gi.repository import AyatanaAppIndicator3 as AppIndicator3, GLib, Gtk
except Exception as exc:
    sys.stderr.write("Mazy OS indicator needs AyatanaAppIndicator3 and GTK 3 (%s)\n" % exc)
    sys.exit(1)


def status():
    try:
        out = subprocess.check_output(["mazzy-panel", "status"], text=True, timeout=8)
    except (OSError, subprocess.SubprocessError) as exc:
        return "Mazzy VPN — %s" % exc
    text = out.strip()
    return text or "Mazzy VPN"


def refresh(indicator):
    indicator.set_label(status(), "")
    return True


def run(cmd):
    subprocess.Popen(["mazzy-panel", cmd])


indicator = AppIndicator3.Indicator.new(
    "mazzy-vpn",
    "network-vpn",
    AppIndicator3.IndicatorCategory.APPLICATION_STATUS,
)
indicator.set_status(AppIndicator3.IndicatorStatus.ACTIVE)
menu = Gtk.Menu()
for label, cmd in (("Connect", "connect"), ("Disconnect", "disconnect")):
    item = Gtk.MenuItem(label=label)
    item.connect("activate", lambda _item, c=cmd: run(c))
    menu.append(item)
menu.show_all()
indicator.set_menu(menu)
refresh(indicator)
GLib.timeout_add_seconds(3, refresh, indicator)
Gtk.main()
