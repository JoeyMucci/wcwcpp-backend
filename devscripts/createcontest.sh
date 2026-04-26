#!/bin/bash
curl -X POST http://localhost:8080/api.v1.ContestService/CreateContest \
  -H "Content-Type: application/json" \
  -H "$HEADER" \
  -d '{
    "title": "My Awesome Contest",
    "groups": [
      {"letter": "A", "countries": [{"code": "A1", "fullName": "Team A1"}, {"code": "A2", "fullName": "Team A2"}, {"code": "A3", "fullName": "Team A3"}, {"code": "A4", "fullName": "Team A4"}]},
      {"letter": "B", "countries": [{"code": "B1", "fullName": "Team B1"}, {"code": "B2", "fullName": "Team B2"}, {"code": "B3", "fullName": "Team B3"}, {"code": "B4", "fullName": "Team B4"}]},
      {"letter": "C", "countries": [{"code": "C1", "fullName": "Team C1"}, {"code": "C2", "fullName": "Team C2"}, {"code": "C3", "fullName": "Team C3"}, {"code": "C4", "fullName": "Team C4"}]},
      {"letter": "D", "countries": [{"code": "D1", "fullName": "Team D1"}, {"code": "D2", "fullName": "Team D2"}, {"code": "D3", "fullName": "Team D3"}, {"code": "D4", "fullName": "Team D4"}]},
      {"letter": "E", "countries": [{"code": "E1", "fullName": "Team E1"}, {"code": "E2", "fullName": "Team E2"}, {"code": "E3", "fullName": "Team E3"}, {"code": "E4", "fullName": "Team E4"}]},
      {"letter": "F", "countries": [{"code": "F1", "fullName": "Team F1"}, {"code": "F2", "fullName": "Team F2"}, {"code": "F3", "fullName": "Team F3"}, {"code": "F4", "fullName": "Team F4"}]},
      {"letter": "G", "countries": [{"code": "G1", "fullName": "Team G1"}, {"code": "G2", "fullName": "Team G2"}, {"code": "G3", "fullName": "Team G3"}, {"code": "G4", "fullName": "Team G4"}]},
      {"letter": "H", "countries": [{"code": "H1", "fullName": "Team H1"}, {"code": "H2", "fullName": "Team H2"}, {"code": "H3", "fullName": "Team H3"}, {"code": "H4", "fullName": "Team H4"}]},
      {"letter": "I", "countries": [{"code": "I1", "fullName": "Team I1"}, {"code": "I2", "fullName": "Team I2"}, {"code": "I3", "fullName": "Team I3"}, {"code": "I4", "fullName": "Team I4"}]},
      {"letter": "J", "countries": [{"code": "J1", "fullName": "Team J1"}, {"code": "J2", "fullName": "Team J2"}, {"code": "J3", "fullName": "Team J3"}, {"code": "J4", "fullName": "Team J4"}]},
      {"letter": "K", "countries": [{"code": "K1", "fullName": "Team K1"}, {"code": "K2", "fullName": "Team K2"}, {"code": "K3", "fullName": "Team K3"}, {"code": "K4", "fullName": "Team K4"}]},
      {"letter": "L", "countries": [{"code": "L1", "fullName": "Team L1"}, {"code": "L2", "fullName": "Team L2"}, {"code": "L3", "fullName": "Team L3"}, {"code": "L4", "fullName": "Team L4"}]}
    ],
    "groupUnlockDate": "2026-06-01T00:00:00Z",
    "groupLockDate": "2026-06-11T00:00:00Z",
    "knockoutUnlockDate": "2026-06-25T00:00:00Z",
    "knockoutLockDate": "2026-06-28T00:00:00Z"
  }'
