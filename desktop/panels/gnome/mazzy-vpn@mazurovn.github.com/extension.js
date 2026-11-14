// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright © 2026 Nik m (@mazurovn). All rights reserved.
//
// GNOME Shell loads this from metadata.json. The label is `mazzy-panel status`.
// Connect and disconnect call the same helper, which elevates only the CLI.

import GLib from 'gi://GLib';
import GObject from 'gi://GObject';
import St from 'gi://St';
import Clutter from 'gi://Clutter';

import {Extension} from 'resource:///org/gnome/shell/extensions/extension.js';
import * as Main from 'resource:///org/gnome/shell/ui/main.js';
import * as PanelMenu from 'resource:///org/gnome/shell/ui/panelMenu.js';

const Indicator = GObject.registerClass(
class Indicator extends PanelMenu.Button {
    _init() {
        super._init(0.0, 'Mazzy VPN');
        this._label = new St.Label({
            text: 'Mazzy VPN',
            y_align: Clutter.ActorAlign.CENTER,
        });
        this.add_child(this._label);
        this.menu.addAction('Connect', () => this._run('connect'));
        this.menu.addAction('Disconnect', () => this._run('disconnect'));
        this._refresh();
        this._timer = GLib.timeout_add_seconds(GLib.PRIORITY_DEFAULT, 3, () => {
            this._refresh();
            return GLib.SOURCE_CONTINUE;
        });
    }

    _refresh() {
        try {
            const [, out] = GLib.spawn_command_line_sync('mazzy-panel status');
            const text = new TextDecoder().decode(out).trim();
            if (text)
                this._label.text = text;
        } catch (e) {
            this._label.text = 'Mazzy VPN';
        }
    }

    _run(cmd) {
        try {
            GLib.spawn_command_line_async(`mazzy-panel ${cmd}`);
        } catch (e) {
            this._label.text = 'Mazzy VPN';
        }
    }

    destroy() {
        if (this._timer) {
            GLib.source_remove(this._timer);
            this._timer = 0;
        }
        super.destroy();
    }
});

export default class MazzyExtension extends Extension {
    enable() {
        this._indicator = new Indicator();
        Main.panel.addToStatusArea('mazzy-vpn', this._indicator);
    }

    disable() {
        if (this._indicator) {
            this._indicator.destroy();
            this._indicator = null;
        }
    }
}
