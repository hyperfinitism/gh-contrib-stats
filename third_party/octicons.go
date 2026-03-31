// SPDX-License-Identifier: Apache-2.0 OR MIT

// Package octicons embeds Primer Octicon SVG files from the primer-octicons
// subdirectory.
//
// The SVG files are sourced from https://github.com/primer/octicons and are
// licensed under the MIT License. See NOTICE.md in the project root for full
// attribution and license text.
package octicons

import "embed"

// FS contains every *.svg file in the primer-octicons subdirectory,
// embedded at compile time.
//
//go:embed primer-octicons/*.svg
var FS embed.FS
