# thanks.dev utils
This repository contains the following tools:
  - auto-boost: used to automatically boost any GH repository that has a "tag-production" topic associated to +2
  - mass-gh-sponsor: used to set up monthly GH sponsorships for all dependencies

## 1. auto-boost
```
utils git:(master) ✗ ./scripts/auto-boost --help
Usage: auto-boost --td-api-url="https://api.thanks.dev" --td-api-key=TD-API-KEY --gh-classic-access-token=GH-ACCESS-TOKEN --entities=ENTITIES,...

Flags:
  -h, --help                                   Show context-sensitive help.
  -v, --version                                Print version and exit.
  -C, --config=FILE                            Config file ($CONFIG_PATH).
      --td-api-url="https://api.thanks.dev"    API path for thanks.dev ($TD_API_URL).
      --td-api-key=TD-API-KEY                  API key for thanks.dev ($TD_API_KEY).
      --gh-classic-access-token=GH-ACCESS-TOKEN
                                               GitHub classis access token with admin:org & user scopes
                                               ($GH_CLASSIC_ACCESS_TOKEN).
      --entities=ENTITIES,...                  The GitHub entities to process sponsorships for. First entity in the list
                                               is considered DEFAULT.

Observability:
  --log-level=info    Log level (trace,debug,info,warning,error,fatal,panic).
  --log-json          Log in JSON format.
```

### Run locally
`. bin/activate-hermit`
`TD_API_KEY=<API_KEY> GH_CLASSIC_ACCESS_TOKEN=<TOKEN> ./scripts/auto-boost --config example.config.json`


## 2. mass-gh-sponsor
```
➜  utils git:(master) ./scripts/mass-gh-sponsor --help
Usage: mass-gh-sponsor --db-path="db.sql" <command>

Flags:
  -h, --help                Show context-sensitive help.
  -v, --version             Print version and exit.
  -C, --config=FILE         Config file ($CONFIG_PATH).
      --db-path="db.sql"    Path to db file ($DB_PATH).

Observability:
  --log-level=info    Log level (trace,debug,info,warning,error,fatal,panic).
  --log-json          Log in JSON format.

Commands:
  import-csv       Import list of donations from csv file.
  dl-repos         Import the user's github repos.
  animate-repos    Animate the sponsorable dependencies for each repo.
  donate           Create the require GitHub sponsorships.

Run "mass-gh-sponsor <command> --help" for more information on a command.
```

### 2.1 Run locally (import from gh)
`. bin/activate-hermit`
`GH_CLASSIC_ACCESS_TOKEN=<TOKEN> ./scripts/mass-gh-sponsor --log-level=debug dl-repos --entities=syntaxfm`
`GH_CLASSIC_ACCESS_TOKEN=<TOKEN> ./scripts/mass-gh-sponsor --log-level=debug animate-repos`
`GH_CLASSIC_ACCESS_TOKEN=<TOKEN> ./scripts/mass-gh-sponsor --log-level=debug donate`

### 2.2 Run locally (import from csv)
`. bin/activate-hermit`
`GH_CLASSIC_ACCESS_TOKEN=<TOKEN> ./scripts/mass-gh-sponsor --log-level=debug import-csv --entity=syntaxfm --file-path=<PATH_TO_CSV_FILE>`
`GH_CLASSIC_ACCESS_TOKEN=<TOKEN> ./scripts/mass-gh-sponsor --log-level=debug donate`

## 3. TD-API-KEY
To obtain a thanks.dev API key, log into thanks.dev and visit the settings screen. The API key configurations are located towards the bottom of the screen.
![image](https://github.com/thnxdev/utils/assets/72539235/610b19f4-2c52-4060-b17f-8f81ba8dbaf7)

## 4. GH-ACCESS-TOKEN
Ensure you create a classic GH access token with `admin:org` and `user` scopes configured. Set a custom expiration date to one day after the last expected donation date.
![image](https://github.com/thnxdev/utils/assets/72539235/a5ffdd99-0db0-4945-a95b-033864c56685)

**Ensure you keep the token stored securely**
Unfortunately, these are the minimum scopes that can create a sponsorship via the GH GraphQL API and they contain write permissions on your account.

