package ai

// thinUniform picks budget indices out of 0..n-1, spread as evenly as integer
// arithmetic allows: the first index is 0, the last is n-1, and each step is
// i*(n-1)/(budget-1). It exists because a video has to be described by a fixed
// number of frames no matter how long it runs, and taking every k-th frame would
// drift away from both ends.
//
// n <= budget returns every index, and a budget of one collapses to the first
// index so the formula never divides by zero.
func thinUniform(n, budget int) []int {
	if n <= 0 || budget <= 0 {
		return []int{}
	}
	if budget == 1 {
		return []int{0}
	}
	if n <= budget {
		idx := make([]int, n)
		for i := range idx {
			idx[i] = i
		}
		return idx
	}
	idx := make([]int, budget)
	for i := range idx {
		idx[i] = i * (n - 1) / (budget - 1)
	}
	return idx
}
