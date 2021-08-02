package testserver_test

import (
	"bytes"
	"encoding/json"
	"net"
	"testing"

	"github.com/rumblefrog/go-a2s"

	"github.com/armsnyder/a2s-exporter/internal/testserver"
)

func TestTestServer_Serve(t *testing.T) {
	type fields struct {
		ServerInfo *a2s.ServerInfo
		PlayerInfo *a2s.PlayerInfo
	}

	tests := []struct {
		name   string
		fields fields
	}{
		{
			name: "empty",
		},
		{
			name: "server info shallow",
			fields: fields{
				ServerInfo: &a2s.ServerInfo{
					Protocol:   1,
					Name:       "foo",
					Map:        "map",
					Folder:     "folder",
					Game:       "game",
					ID:         234,
					Players:    22,
					MaxPlayers: 33,
					Bots:       4,
					ServerType: a2s.ServerType_Dedicated,
					ServerOS:   a2s.ServerOS_Mac,
					Visibility: true,
					VAC:        true,
					Version:    "ver",
				},
			},
		},
		{
			name: "server info the ship full",
			fields: fields{
				ServerInfo: &a2s.ServerInfo{
					Protocol:   1,
					Name:       "foo",
					Map:        "map",
					Folder:     "folder",
					Game:       "game",
					ID:         uint16(a2s.App_TheShip),
					Players:    22,
					MaxPlayers: 33,
					Bots:       4,
					ServerType: a2s.ServerType_Dedicated,
					ServerOS:   a2s.ServerOS_Mac,
					Visibility: true,
					VAC:        true,
					TheShip: &a2s.TheShipInfo{
						Mode:      a2s.TheShipMode_Elimination,
						Witnesses: 3,
						Duration:  45,
					},
					Version: "ver",
					ExtendedServerInfo: &a2s.ExtendedServerInfo{
						Port:     4572,
						SteamID:  2367893276,
						Keywords: "abc",
						GameID:   12345,
					},
					SourceTV: &a2s.SourceTVInfo{
						Port: 3463,
						Name: "tv",
					},
				},
			},
		},
		{
			name: "player info",
			fields: fields{
				PlayerInfo: &a2s.PlayerInfo{
					Count: 2,
					Players: []*a2s.Player{
						{
							Index:    0,
							Name:     "jon",
							Score:    4,
							Duration: 234,
						},
						{
							Index:    1,
							Name:     "alice",
							Score:    3457,
							Duration: 4564,
						},
					},
				},
			},
		},
		{
			name: "player info the ship",
			fields: fields{
				ServerInfo: &a2s.ServerInfo{
					ID: uint16(a2s.App_TheShip),
					TheShip: &a2s.TheShipInfo{
						Mode: a2s.TheShipMode_Duel,
					},
				},
				PlayerInfo: &a2s.PlayerInfo{
					Count: 2,
					Players: []*a2s.Player{
						{
							Index:    0,
							Name:     "jon",
							Score:    4,
							Duration: 234,
							TheShip: &a2s.TheShipPlayer{
								Deaths: 23,
								Money:  3456,
							},
						},
						{
							Index:    1,
							Name:     "alice",
							Score:    3457,
							Duration: 4564,
							TheShip: &a2s.TheShipPlayer{
								Deaths: 345,
								Money:  123,
							},
						},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test server listener.
			conn, err := net.ListenUDP("udp", nil)
			if err != nil {
				t.Fatal(err)
			}
			defer conn.Close()

			// Initialize TestServer.
			srv := &testserver.TestServer{
				ServerInfo: tt.fields.ServerInfo,
				PlayerInfo: tt.fields.PlayerInfo,
			}

			// Serve in background.
			go func() {
				_ = srv.Serve(conn)
			}()

			// Construct the client.
			client, err := a2s.NewClient(conn.LocalAddr().String())
			if err != nil {
				t.Fatal(err)
			}

			// Query the server info and check that it matches how the TestServer was initialized.
			serverInfo, err := client.QueryInfo()
			if err != nil {
				t.Fatalf("Unexpected error while querying server info: %v", err)
			} else {
				serverInfo.EDF = 0
				want := &a2s.ServerInfo{}
				testJSONCopy(t, want, tt.fields.ServerInfo)
				testAssertJSONEqual(t, want, serverInfo)
			}

			// Re-construct the client with the app ID from the server info (required for The Ship).
			client, err = a2s.NewClient(conn.LocalAddr().String(), a2s.SetAppID(int32(serverInfo.ID)))
			if err != nil {
				t.Fatal(err)
			}

			// Query the player info and check that it matches how the TestServer was initialized.
			if playerInfo, err := client.QueryPlayer(); err != nil {
				t.Errorf("Unexpected error while querying player info: %v", err)
			} else {
				want := &a2s.PlayerInfo{}
				testJSONCopy(t, want, tt.fields.PlayerInfo)
				testAssertJSONEqual(t, want, playerInfo)
			}
		})
	}
}

// testJSONCopy unmarshalls into dest using the JSON encoding of src.
func testJSONCopy(t *testing.T, dest, src interface{}) {
	t.Helper()
	b, err := json.Marshal(src)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(b, dest); err != nil {
		t.Fatal(err)
	}
}

// testAssertJSONEqual checks for JSON-encoded equality.
func testAssertJSONEqual(t *testing.T, want, got interface{}) {
	t.Helper()
	wantBytes, err := json.MarshalIndent(want, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	gotBytes, err := json.MarshalIndent(got, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(gotBytes, wantBytes) {
		t.Errorf("not equal\n\nwanted:\n%s\n\ngot:\n%s\n", string(wantBytes), string(gotBytes))
	}
}
