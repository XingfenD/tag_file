package ai

// mergeRawTags folds the raw terms of every frame of one file into the single list
// that feeds the review queue. The same scene stays on screen across neighbouring
// frames, so the model says the same thing about it more than once; that repetition
// must not turn into one queue row per frame.
//
// Three rules, applied together:
//   - an exact (Category, Tag, Value) triple is kept once, at its first occurrence,
//     and the leftover order of first occurrences is preserved;
//   - if any row of a (Category, Tag) pair carries a value, the valueless rows of
//     that pair are dropped — whatever their position. A frame that only names the
//     scene and a frame that names it *with* a value describe the same tag, and the
//     bare row adds nothing a human can act on; dropping it before the review queue
//     is what keeps 猫/橘色 from arriving next to 猫/-;
//   - different values of one tag are all kept. 黑色裤子 and 白色裤子 may both be in
//     the video, and deciding that only one is meant is the reviewer's call, not this
//     function's — collapsing them here would silently lose a suggestion.
//
// An empty input yields an empty (non-nil) slice so the caller can range over the
// result and marshal it without a null.
func mergeRawTags(raws []RawTag) []RawTag {
	// First pass: which (Category, Tag) pairs are already known to carry a value.
	// Collected up front because rule 2 looks at the whole file, not at the frames
	// that came before — an empty row in frame 1 still loses to a valued row from
	// frame 9.
	valued := make(map[[2]string]bool, len(raws))
	for _, raw := range raws {
		if raw.Value != "" {
			valued[[2]string{raw.Category, raw.Tag}] = true
		}
	}

	// Second pass: walk in order, keeping the first occurrence of each triple and
	// skipping the valueless rows of a pair that is represented by a valued one.
	merged := make([]RawTag, 0, len(raws))
	seen := make(map[[3]string]bool, len(raws))
	for _, raw := range raws {
		triple := [3]string{raw.Category, raw.Tag, raw.Value}
		if seen[triple] {
			continue
		}
		if raw.Value == "" && valued[[2]string{raw.Category, raw.Tag}] {
			continue
		}
		seen[triple] = true
		merged = append(merged, raw)
	}
	return merged
}
