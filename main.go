package main

import (
	"fmt"
	"slices"

	gallerysync "github.com/clouderhem/misync/misync/pull/gallery"
)

func main() {
	Test()
}

func Test() {
	var ts []gallerysync.Timeline

	ts = append(ts,
		gallerysync.Timeline{
			StartDate: 1,
			EndDate:   1,
			Count:     2,
		},
		gallerysync.Timeline{
			StartDate: 2,
			EndDate:   2,
			Count:     2,
		},
		gallerysync.Timeline{
			StartDate: 3,
			EndDate:   3,
			Count:     1,
		},
		gallerysync.Timeline{
			StartDate: 4,
			EndDate:   4,
			Count:     1,
		},
	)

	slices.SortFunc(ts, func(a, b gallerysync.Timeline) int {
		return a.StartDate - b.StartDate
	})

	var sum, lastDate = 0, ts[0].StartDate
	var result []gallerysync.Timeline
	for i := range ts {
		if sum+ts[i].Count > 100 {
			var t gallerysync.Timeline
			if i == 0 {
				t = gallerysync.Timeline{StartDate: lastDate, EndDate: lastDate, Count: sum}
			} else {
				t = gallerysync.Timeline{StartDate: lastDate, EndDate: ts[i-1].EndDate, Count: sum}
			}
			result = append(result, t)
			sum = 0
			lastDate = ts[i].StartDate
		}
		sum += ts[i].Count
		if i == len(ts)-1 {
			result = append(result, gallerysync.Timeline{StartDate: lastDate, EndDate: ts[i].EndDate, Count: sum})

		}
	}
	fmt.Println(result)
}
