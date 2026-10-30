package router

import (
	"errors"

	"github.com/dilly3/edehson-league/models"
)

var errScheduleConflict = errors.New("existing fixtures cannot fit the league schedule")

func missingFixtures(tags []string, existing []models.Fixture) ([]models.Fixture, error) {
	if len(tags) < 2 || len(tags) > 4 {
		return nil, errScheduleConflict
	}
	teamIndex := make(map[string]int)
	for i, tag := range tags {
		if _, duplicate := teamIndex[tag]; duplicate {
			return nil, errScheduleConflict
		}
		teamIndex[tag] = i
	}

	// Two teams need two rounds; three or four teams need six.
	rounds := 6
	if len(tags) == 2 {
		rounds = 2
	}
	used := make([]int, rounds)
	seen := make(map[[2]string]bool)
	for _, f := range existing {
		h, homeOK := teamIndex[f.HomeTeamTag]
		a, awayOK := teamIndex[f.AwayTeamTag]
		key := [2]string{f.HomeTeamTag, f.AwayTeamTag}
		if !homeOK || !awayOK || h == a || f.Matchday < 1 || f.Matchday > rounds || f.DeletedAt != 0 || seen[key] {
			return nil, errScheduleConflict
		}
		mask := (1 << h) | (1 << a)
		if used[f.Matchday-1]&mask != 0 {
			return nil, errScheduleConflict
		}
		used[f.Matchday-1] |= mask
		seen[key] = true
	}

	pending := make([]models.Fixture, 0)
	for _, home := range tags {
		for _, away := range tags {
			if home != away && !seen[[2]string{home, away}] {
				pending = append(pending, models.Fixture{
					HomeTeamTag: home, AwayTeamTag: away, Status: "scheduled",
				})
			}
		}
	}

	// Try the most constrained remaining fixture first.
	var place func(int) bool
	place = func(start int) bool {
		if start == len(pending) {
			return true
		}
		best, fewest := start, rounds+1
		for i := start; i < len(pending); i++ {
			f := pending[i]
			mask := (1 << teamIndex[f.HomeTeamTag]) | (1 << teamIndex[f.AwayTeamTag])
			available := 0
			for _, occupied := range used {
				if occupied&mask == 0 {
					available++
				}
			}
			if available < fewest {
				best, fewest = i, available
			}
		}
		if fewest == 0 {
			return false
		}
		pending[start], pending[best] = pending[best], pending[start]
		f := pending[start]
		mask := (1 << teamIndex[f.HomeTeamTag]) | (1 << teamIndex[f.AwayTeamTag])
		for day := range used {
			if used[day]&mask != 0 {
				continue
			}
			used[day] |= mask
			pending[start].Matchday = day + 1
			if place(start + 1) {
				return true
			}
			used[day] &^= mask
		}
		pending[start], pending[best] = pending[best], pending[start]
		return false
	}
	if !place(0) {
		return nil, errScheduleConflict
	}
	return pending, nil
}
