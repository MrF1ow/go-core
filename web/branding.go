package web

import (
	"fmt"
	"math"
	"strconv"
)

// ParseHexColor parses a CSS hex color string (3-digit or 6-digit, with leading #)
// into its red, green, and blue uint8 components. It is case-insensitive.
func ParseHexColor(hex string) (r, g, b uint8, err error) {
	if len(hex) == 0 || hex[0] != '#' {
		return 0, 0, 0, fmt.Errorf("invalid hex color: %q", hex)
	}
	hex = hex[1:]

	switch len(hex) {
	case 3:
		rr, err1 := strconv.ParseUint(string(hex[0])+string(hex[0]), 16, 8)
		gg, err2 := strconv.ParseUint(string(hex[1])+string(hex[1]), 16, 8)
		bb, err3 := strconv.ParseUint(string(hex[2])+string(hex[2]), 16, 8)
		if err1 != nil || err2 != nil || err3 != nil {
			return 0, 0, 0, fmt.Errorf("invalid hex color: #%s", hex)
		}
		return uint8(rr), uint8(gg), uint8(bb), nil
	case 6:
		rr, err1 := strconv.ParseUint(hex[0:2], 16, 8)
		gg, err2 := strconv.ParseUint(hex[2:4], 16, 8)
		bb, err3 := strconv.ParseUint(hex[4:6], 16, 8)
		if err1 != nil || err2 != nil || err3 != nil {
			return 0, 0, 0, fmt.Errorf("invalid hex color: #%s", hex)
		}
		return uint8(rr), uint8(gg), uint8(bb), nil
	default:
		return 0, 0, 0, fmt.Errorf("invalid hex color: #%s", hex)
	}
}

// RelativeLuminance computes the WCAG 2.1 relative luminance of an sRGB color.
// Returns a value in [0, 1] where 0 is black and 1 is white.
func RelativeLuminance(r, g, b uint8) float64 {
	rl := linearize(float64(r) / 255.0)
	gl := linearize(float64(g) / 255.0)
	bl := linearize(float64(b) / 255.0)
	return 0.2126*rl + 0.7152*gl + 0.0722*bl
}

// linearize converts an sRGB channel value to linear light.
func linearize(c float64) float64 {
	if c <= 0.04045 {
		return c / 12.92
	}
	return math.Pow((c+0.055)/1.055, 2.4)
}

// AutoSidebarTextColor returns "#ffffff" for dark sidebar backgrounds and
// "#212529" for light ones, based on WCAG relative luminance. Falls back to
// "#ffffff" if sidebarHex cannot be parsed.
func AutoSidebarTextColor(sidebarHex string) string {
	r, g, b, err := ParseHexColor(sidebarHex)
	if err != nil {
		return "#ffffff"
	}
	if RelativeLuminance(r, g, b) > 0.5 {
		return "#212529"
	}
	return "#ffffff"
}

// ResolvedBranding holds the fully resolved branding values ready for template
// rendering. Fields remain empty when the consumer does not configure them.
type ResolvedBranding struct {
	OrgName          string
	LogoURL          string
	PrimaryColor     string
	SecondaryColor   string
	BorderRadius     string
	SidebarColor     string
	SidebarTextColor string
}

// ResolveBranding applies precedence rules to produce a ResolvedBranding:
//   - logoServeURL overrides logoURL when non-empty (served upload takes priority)
//   - sidebarTextColor is used as-is when set; otherwise it is auto-derived from
//     sidebarColor via WCAG luminance; if sidebarColor is also empty the field is
//     left blank for the template to apply its own default.
func ResolveBranding(orgName, logoURL, primaryColor, secondaryColor, borderRadius, sidebarColor, sidebarTextColor, logoServeURL string) ResolvedBranding {
	resolved := ResolvedBranding{
		OrgName:        orgName,
		LogoURL:        logoURL,
		PrimaryColor:   primaryColor,
		SecondaryColor: secondaryColor,
		BorderRadius:   borderRadius,
		SidebarColor:   sidebarColor,
	}

	if logoServeURL != "" {
		resolved.LogoURL = logoServeURL
	}

	if sidebarTextColor != "" {
		resolved.SidebarTextColor = sidebarTextColor
	} else if sidebarColor != "" {
		resolved.SidebarTextColor = AutoSidebarTextColor(sidebarColor)
	}

	return resolved
}
