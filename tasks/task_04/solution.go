package main

type Stats struct {
	Count         int
	Sum, Min, Max int64
}

func Calc(nums []int64) Stats {
	n := len(nums)

	if n < 2 {
		return Stats{}
	}

	firstDiff := nums[1] - nums[0]
	
	sum := firstDiff
	min := firstDiff
	max := firstDiff
	count := 1

	for i := 2; i < n; i++ {
		count++
		result := nums[i] - nums[i - 1]
		sum += result

		if result > max {
			max = result
		}

		if result < min {
			min = result
		}
	}

	return Stats{ Count: count, Sum: sum, Max: max, Min: min }
}
