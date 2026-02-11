package hac

import (
	"strconv"
	"strings"
)

// mediaRange represents a parsed media range from an Accept header.
type mediaRange struct {
	typ     string
	subtype string
	quality float64
}

// parseAccept parses an HTTP Accept header into a slice of media ranges sorted
// by descending quality factor.
func parseAccept(header string) []mediaRange {
	if header == "" {
		return nil
	}
	parts := strings.Split(header, ",")
	ranges := make([]mediaRange, 0, len(parts))
	for _, part := range parts {
		mr := parseMediaRange(strings.TrimSpace(part))
		if mr.typ != "" {
			ranges = append(ranges, mr)
		}
	}
	return ranges
}

// parseMediaRange parses a single media range entry like "application/json;q=0.8".
func parseMediaRange(s string) mediaRange {
	mr := mediaRange{quality: 1.0}

	// Split off parameters
	params := strings.Split(s, ";")
	mediaType := strings.TrimSpace(params[0])

	slash := strings.IndexByte(mediaType, '/')
	if slash < 0 {
		return mediaRange{}
	}
	mr.typ = strings.TrimSpace(mediaType[:slash])
	mr.subtype = strings.TrimSpace(mediaType[slash+1:])

	if mr.typ == "" || mr.subtype == "" {
		return mediaRange{}
	}

	// Parse quality factor
	for _, param := range params[1:] {
		param = strings.TrimSpace(param)
		if strings.HasPrefix(param, "q=") || strings.HasPrefix(param, "Q=") {
			if q, err := strconv.ParseFloat(param[2:], 64); err == nil {
				mr.quality = q
			}
		}
	}

	return mr
}

// wantsHAC returns true if the Accept header includes the HAC media type
// with a quality factor > 0.
func wantsHAC(accept string) bool {
	for _, mr := range parseAccept(accept) {
		if matchesHAC(mr) && mr.quality > 0 {
			return true
		}
	}
	return false
}

// hacIsOnlyAcceptable returns true if the Accept header indicates HAC is the
// only acceptable type (no wildcards, no other types with q > 0).
func hacIsOnlyAcceptable(accept string) bool {
	ranges := parseAccept(accept)
	if len(ranges) == 0 {
		return false
	}
	hasHAC := false
	for _, mr := range ranges {
		if matchesHAC(mr) {
			if mr.quality > 0 {
				hasHAC = true
			}
			continue
		}
		// Any other type with q > 0 means HAC isn't the only option
		if mr.quality > 0 {
			return false
		}
	}
	return hasHAC
}

// matchesHAC returns true if the media range matches the HAC media type,
// including wildcard matches.
func matchesHAC(mr mediaRange) bool {
	if mr.typ == "application" && mr.subtype == "vnd.hac+json" {
		return true
	}
	return false
}
