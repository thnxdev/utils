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
Usage: mass-gh-sponsor --db-path="db.sql" --gh-classic-access-token=GH-ACCESS-TOKEN --entities=ENTITIES,...

Flags:
  -h, --help                     Show context-sensitive help.
  -v, --version                  Print version and exit.
  -C, --config=FILE              Config file ($CONFIG_PATH).
      --db-path="db.sql"         Path to db file ($DB_PATH).
      --gh-classic-access-token=GH-ACCESS-TOKEN
                                 GitHub classis access token with admin:org & user scopes ($GH_CLASSIC_ACCESS_TOKEN).
      --entities=ENTITIES,...    The GitHub entities to process sponsorships for. First entity in the list is considered
                                 DEFAULT.
      --sponsor-amount=1         The amount to donate to each dependency

Observability:
  --log-level=info    Log level (trace,debug,info,warning,error,fatal,panic).
  --log-json          Log in JSON format.

wkr-donate
  --wkr-donate-disabled             Flag to disable worker
  --wkr-donate-work-interval=10s    Sleep duration per run while processing

wkr-entities
  --wkr-entities-disabled             Flag to disable worker
  --wkr-entities-work-interval=10s    Sleep duration per run while processing

wkr-repos
  --wkr-repos-disabled             Flag to disable worker
  --wkr-repos-work-interval=10s    Sleep duration per run while processing
```

### 2.1 Run locally
`. bin/activate-hermit`
`GH_CLASSIC_ACCESS_TOKEN=<TOKEN> ./scripts/mass-gh-sponsor --config example.config.json`

### 2.2 Docker & docker compose
A sample docker-compose.yml is provided to run this project.
```
cp example.env .env # update the latest api keys & access tokens
cp example.config.json config.json # insert the github slugs of the user accounts / orgs
docker compose --env-file .env up -d mass-gh-sponsor
```

## 3. TD-API-KEY
To obtain a thanks.dev API key, log into thanks.dev and visit the settings screen. The API key configurations are located towards the bottom of the screen.
![image](https://github.com/thnxdev/utils/assets/72539235/9ff22805-164d-47ba-a71a-07e7cb6d832a)


## 4. GH-ACCESS-TOKEN
Ensure you create a classic GH access token with `admin:org` and `user` scopes configured. Set a custom expiration date to one day after the last expected donation date.
![image](https://github.com/thnxdev/utils/assets/72539235/69f248a8-2351-471e-84d5-43eeba9d3f5f)

**Ensure you keep the token stored securely**
Unfortunately, these are the minimum scopes that can create a sponsorship via the GH GraphQL API and they contain write permissions on your account.

