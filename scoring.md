# WCWCPP Scoring

Scoring for the WCWCPP (Wacky & Cool World Cup Prediction Platform) is divided into two stages: 
1. Group Stage Scoring
2. Knockout Stage Scoring


## Group Stage Scoring

For all 12 groups in the world cup, each player will rank the four teams in order from 1-4. The teams they rank get multipliers as follows

| Rank | Multiplier |
|------|------------|
| 1 | 3 |
| 2 | 2 |
| 3 | 1 |
| 4 | 0 |

And teams earn points according to their placement:

| Rank | Points |
|------|--------|
| 1 | 10 |
| 2 | 6 |
| 3 | 3 |
| 4 | 1 |

Therefore, the maximum possible score for the group stage is when a player ranks the teams in the same order as they finish, and each match is a win for the higher ranked team. In this case, the score would be:

10 * 3 + 6 * 2 + 3 * 1 + 1 * 0 = 30 + 12 + 3 + 0 = 45 points

Additionally, players can predict whether the third place team will advance to the knockout stage. This is a binary prediction, so there are 2 possible outcomes. This should be a simple boolean true/false, and players will earn 5 points if they correctly predict whether the third place team will advance to the knockout stage.

This means there are a maximum of 50 points possible per group in the group stage. Since there are 12 groups, the maximum possible points in the group stage is 50 * 12 = 600 points.

Points for the group stage will be calculated on a once per contest basis, triggered by the conclusion of a particular group stage's matches. There will also be an additional superadmin endpoint to trigger the third place bonus points for all groups.

## Knockout Stage Scoring

Users gets points for predicting a team will make it to a given round.

| Round | Points |
|-------|--------|
| Round of 16 | 15 |
| Quarterfinals | 20 |
| Semifinals | 25 |
| Final | 30 |
| Winner | 35 |
| Third Place | 5 |

The score for a perfect bracket with all 32 picks correct is:

16 * 15 + 8 * 20 + 4 * 25 + 2 * 30 + 1 * 35 + 1 * 5 = 240 + 160 + 100 + 60 + 35 + 5 = 600 points.

Knockout points will be awarded on every knockout match. 

## Awards

The max amount of points a player can earn in total is 1200 points. 

There will be 3 winners for distinct users. The overall winner will be the user with the most points. The next two winners will be the next user with most points in the knockout stage and the next user with most points in the group stage.
