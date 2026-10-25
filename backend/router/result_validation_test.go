package router

import (
	"strings"
	"testing"

	"github.com/dilly3/edehson-league/models"
)

func TestValidateResult(t *testing.T) {
	const (
		homePlayer = "11111111-1111-4111-8111-111111111111"
		awayPlayer = "22222222-2222-4222-8222-222222222222"
		outsider   = "33333333-3333-4333-8333-333333333333"
	)

	fixture := models.Fixture{
		Model: models.Model{
			ID: "44444444-4444-4444-8444-444444444444",
		},
		HomeTeamTag: "EDL001",
		AwayTeamTag: "EDL002",
	}

	roster := map[string]string{
		homePlayer: "EDL001",
		awayPlayer: "EDL002",
	}

	event := func(player, kind string, minute int) eventInput {
		return eventInput{
			PlayerID: player,
			Type:     kind,
			Minute:   &minute,
		}
	}

	result := func(home, away int, events ...eventInput) resultInput {
		revision := 0

		return resultInput{
			HomeScore: &home,
			AwayScore: &away,
			Revision:  &revision,
			Events:    events,
		}
	}

	tests := []struct {
		name    string
		input   resultInput
		wantErr string
	}{
		{
			name:  "scoreless draw without events",
			input: result(0, 0),
		},
		{
			name: "goal by each team",
			input: result(1, 1,
				event(homePlayer, models.EventGoal, 10),
				event(awayPlayer, models.EventGoal, 20),
			),
		},
		{
			name: "own goal credits opposing team",
			input: result(0, 1,
				event(homePlayer, models.EventOwnGoal, 15),
			),
		},
		{
			name: "multiple goals by same player",
			input: result(2, 0,
				event(homePlayer, models.EventGoal, 10),
				event(homePlayer, models.EventGoal, 30),
			),
		},
		{
			name: "two yellow cards are allowed",
			input: result(0, 0,
				event(homePlayer, models.EventYellowCard, 20),
				event(homePlayer, models.EventYellowCard, 60),
			),
		},
		{
			name: "third yellow card is rejected",
			input: result(0, 0,
				event(homePlayer, models.EventYellowCard, 20),
				event(homePlayer, models.EventYellowCard, 40),
				event(homePlayer, models.EventYellowCard, 60),
			),
			wantErr: "more than two yellow cards",
		},
		{
			name: "second red card is rejected",
			input: result(0, 0,
				event(homePlayer, models.EventRedCard, 20),
				event(homePlayer, models.EventRedCard, 60),
			),
			wantErr: "more than one red card",
		},
		{
			name: "straight red after first yellow",
			input: result(0, 0,
				event(homePlayer, models.EventYellowCard, 20),
				event(homePlayer, models.EventRedCard, 60),
			),
		},
		{
			name: "events with unknown minutes",
			input: result(1, 0,
				eventInput{
					PlayerID: homePlayer,
					Type:     models.EventGoal,
					Minute:   nil,
				},
			),
		},
		{
			name: "same minute follows array order",
			input: result(0, 0,
				event(homePlayer, models.EventYellowCard, 60),
			),
		},
		{
			name:    "negative score",
			input:   result(-1, 0),
			wantErr: "scores cannot be negative",
		},
		{
			name:    "score without matching goals",
			input:   result(1, 0),
			wantErr: "goal events produce",
		},
		{
			name: "invalid player UUID",
			input: result(1, 0,
				event("invalid", models.EventGoal, 10),
			),
			wantErr: "invalid player ID",
		},
		{
			name: "player outside participating rosters",
			input: result(1, 0,
				event(outsider, models.EventGoal, 10),
			),
			wantErr: "participating team",
		},
		{
			name: "negative event minute",
			input: result(1, 0,
				event(homePlayer, models.EventGoal, -1),
			),
			wantErr: "minute cannot be negative",
		},
		{
			name: "events out of order",
			input: result(1, 1,
				event(homePlayer, models.EventGoal, 40),
				event(awayPlayer, models.EventGoal, 20),
			),
			wantErr: "match order",
		},
		{
			name: "goal after red card is allowed",
			input: result(1, 0,
				event(homePlayer, models.EventRedCard, 20),
				event(homePlayer, models.EventGoal, 60),
			),
		},
		{
			name: "unknown event type",
			input: result(0, 0,
				event(homePlayer, "assist", 20),
			),
			wantErr: "unsupported event type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			events, err := validateResult(fixture, tt.input, roster)

			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("expected error containing %q", tt.wantErr)
				}

				if !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf(
						"expected error containing %q; got %q",
						tt.wantErr,
						err.Error(),
					)
				}

				return
			}

			if err != nil {
				t.Fatalf("unexpected validation error: %v", err)
			}

			if len(events) != len(tt.input.Events) {
				t.Fatalf(
					"expected %d events; got %d",
					len(tt.input.Events),
					len(events),
				)
			}

			for i, saved := range events {
				source := tt.input.Events[i]

				if saved.FixtureID != fixture.ID {
					t.Errorf("event %d: incorrect fixture ID", i+1)
				}

				if saved.PlayerID == nil || *saved.PlayerID != source.PlayerID {
					t.Errorf("event %d: incorrect player ID", i+1)
				}

				if saved.TeamTag != roster[source.PlayerID] {
					t.Errorf("event %d: incorrect player team tag", i+1)
				}

				if saved.Type != source.Type {
					t.Errorf("event %d: incorrect event type", i+1)
				}

				if (saved.Minute == nil) != (source.Minute == nil) {
					t.Errorf("event %d: incorrect minute presence", i+1)
				} else if saved.Minute != nil &&
					*saved.Minute != *source.Minute {
					t.Errorf("event %d: incorrect minute", i+1)
				}
			}
		})
	}

	t.Run("required fields", func(t *testing.T) {
		tests := []struct {
			name  string
			input resultInput
		}{
			{
				name: "missing home score",
				input: func() resultInput {
					in := result(0, 0)
					in.HomeScore = nil
					return in
				}(),
			},
			{
				name: "missing away score",
				input: func() resultInput {
					in := result(0, 0)
					in.AwayScore = nil
					return in
				}(),
			},
			{
				name: "missing revision",
				input: func() resultInput {
					in := result(0, 0)
					in.Revision = nil
					return in
				}(),
			},
			{
				name: "negative revision",
				input: func() resultInput {
					in := result(0, 0)
					negative := -1
					in.Revision = &negative
					return in
				}(),
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				if _, err := validateResult(
					fixture,
					tt.input,
					roster,
				); err == nil {
					t.Fatal("expected validation to reject the request")
				}
			})
		}
	})
}

func TestOptionalScorers(t *testing.T) {
	fixture := models.Fixture{HomeTeamTag: "EDL001", AwayTeamTag: "EDL002"}
	player := "11111111-1111-4111-8111-111111111111"
	roster := map[string]string{player: "EDL001"}
	one, zero, revision := 1, 0, 0
	input := resultInput{HomeScore: &one, AwayScore: &one, Revision: &revision, Events: []eventInput{{Type: models.EventGoal, TeamTag: "EDL001"}, {Type: models.EventOwnGoal, TeamTag: "EDL001"}}}
	events, err := validateResult(fixture, input, roster)
	if err != nil {
		t.Fatal(err)
	}
	for _, event := range events {
		if event.PlayerID != nil || event.TeamTag != "EDL001" {
			t.Fatal("unnamed goal assigned to a player or wrong team")
		}
	}
	for _, kind := range []string{models.EventYellowCard, models.EventRedCard} {
		input.HomeScore = &zero
		input.AwayScore = &zero
		input.Events = []eventInput{{Type: kind, TeamTag: "EDL001"}}
		if _, err := validateResult(fixture, input, roster); err == nil {
			t.Fatal("unnamed card accepted")
		}
	}
	input.HomeScore = &one
	input.Events = []eventInput{{Type: models.EventGoal, TeamTag: "OTHER"}}
	if _, err := validateResult(fixture, input, roster); err == nil {
		t.Fatal("nonparticipating team accepted")
	}
	input.Events = []eventInput{{Type: models.EventGoal, TeamTag: "EDL002", PlayerID: player}}
	if _, err := validateResult(fixture, input, roster); err == nil {
		t.Fatal("mismatched player team accepted")
	}
	input.Events = []eventInput{{Type: models.EventGoal, TeamTag: "EDL001"}, {Type: models.EventGoal, TeamTag: "EDL001"}}
	if _, err := validateResult(fixture, input, roster); err == nil {
		t.Fatal("score mismatch accepted")
	}
}
