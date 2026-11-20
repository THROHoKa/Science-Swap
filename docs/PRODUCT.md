# Science Swap

Science Swap helps people find a partner at another university for a semester exchange. Anyone can use the service, regardless of academic role or title.

## Core workflow

1. Create a profile with your university and ISCED-F field of study.
2. Select one or more preferred destination regions. You can leave your available dates open.
3. Review candidates at other universities. Suggestions use compatible fields of study, overlapping availability, and each participant's destination preferences.
4. Express interest in a candidate. A match forms when both people express interest.
5. Contact a mutual match by email. A participant can send a message through the service and choose which personal details to include, including whether to share a reply address. The recipient can make their own sharing choices when responding.

## MVP features

- Accounts and exchange profiles with open eligibility.
- University lookup and bilingual ISCED-F field labels.
- Availability periods and multiple preferred destination regions.
- Candidate suggestions, expressions of interest, and mutual matches.
- Email contact after a mutual match, with email address sharing opt-in.
- German and English interfaces.

## Data model sketch

Application tables use the `SSwap` prefix.

| Entity | Purpose |
| --- | --- |
| Account | Identity and private email address |
| Exchange profile | Account, university, and ISCED-F field |
| University | Name and location for lookup and regional matching |
| Availability | Profile and available date period stored as a PostgreSQL `daterange` |
| Destination preference | One row per profile and preferred region |
| Interest | One profile's expression of interest in another profile |
| Match | Pair of profiles with reciprocal interest |

Suggestions default to the same three-digit ISCED-F field. People who leave their dates open can match across all time intervals. With a specific time interval selected, PostgreSQL's `&&` operator checks date overlap and users can require one, three, or six shared months.

During the email-entry phase, an unverified account is accessible through its current browser session. Contact requests are saved as interest. Email verification and delivery are required before the email-contact step is enabled.

## Out of scope for the MVP

- Home and course swaps.
- A browsable profile gallery, profile photos, and teaching histories.
- Household-size and academic-role filters.
- Coordinate-based radius search.
