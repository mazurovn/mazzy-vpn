// SPDX-License-Identifier: PolyForm-Noncommercial-1.0.0
// Copyright © 2026 Nik m (@mazurovn). All rights reserved.

// Package integration contains end-to-end parity tests that wire the real
// mazzy-core packages together (connect + health + livetest + verify +
// bootrecovery + state + lock) at their true interfaces, with only the kernel
// boundary (netexec.Runner) faked. This is backlog C2-9: prove the packages
// compose into the bash CLI's guard/lease/recovery behaviors.
//
// It has no production code; it exists so the composition itself is tested and
// interface drift between packages fails the build.
package integration
