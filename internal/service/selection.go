package service

func selectWeighted(weights []float64, count int, next func() float64) []int {
	if count > len(weights) {
		count = len(weights)
	}
	if count <= 0 || len(weights) == 0 {
		return nil
	}
	if count == len(weights) {
		indexes := make([]int, count)
		for i := range indexes {
			indexes[i] = i
		}
		return indexes
	}

	available := make([]int, len(weights))
	for i := range weights {
		available[i] = i
	}
	selected := make([]int, 0, count)
	for len(selected) < count && len(available) > 0 {
		total := 0.0
		for _, index := range available {
			total += weights[index]
		}
		target := next() * total
		pick := len(available) - 1
		cumulative := 0.0
		for i, index := range available {
			cumulative += weights[index]
			if target < cumulative {
				pick = i
				break
			}
		}
		selected = append(selected, available[pick])
		available = append(available[:pick], available[pick+1:]...)
	}
	return selected
}
