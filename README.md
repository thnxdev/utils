# wkr-gh-sponsor

This project is a standalone worker that continuosly:
  - Checks https://api.thanks.dev/v1/deps to obtain the latest list of dependencies;
  - Queries GH GraphQL API to determine if a dependency has GH sponsors enabled;
  - One the first day of every month does a `createSponsorship` GH GraphQL mutation;

### Command line flags
```
wkr-gh-sponsor🐚 ➜  wkr-gh-sponsor git:(master) ./scripts/wkr-gh-sponsor --help
Usage: wkr-gh-sponsor --db-path="db.sql" --td-api-key=TD-API-KEY --gh-classic-access-token=GH-ACCESS-TOKEN --entities=ENTITIES,...

Flags:
  -h, --help                     Show context-sensitive help.
  -v, --version                  Print version and exit.
  -C, --config=FILE              Config file ($CONFIG_PATH).
  -e, --env=local                Environment (local,prod) ($APP_ENV).
      --db-path="db.sql"         Path to db file.
      --td-api-key=TD-API-KEY    Api key for thanks.dev ($TD_API_KEY).
      --gh-classic-access-token=GH-ACCESS-TOKEN
                                 GitHub classis access token with admin:org & user scopes ($GH_CLASSIC_ACCESS_TOKEN).
      --entities=ENTITIES,...    The GitHub entities to process sponsorships for. First entity in the list is
                                 considered DEFAULT.

Observability:
  --log-level=info    Log level (trace,debug,info,warning,error,fatal,panic).
  --log-json          Log in JSON format.
```

### Docker & docker compose
A sample docker-compose.yml is provided to run this project.
```
cp example.env .env # update the latest api keys & access tokens
cp example.config.json config.json # insert the github slugs of the user accounts / orgs
docker compose --env-file .env up -d wkr-gh-sponsor
```

### TD-API-KEY
To obtain a thanks.dev API key, log into thanks.dev and visit the settings screen. The API key configurations are located towards the bottom of the screen.
![image](https://github.com/thnxdev/wkr-gh-sponsor/assets/72539235/9ff22805-164d-47ba-a71a-07e7cb6d832a)

### GH-ACCESS-TOKEN
Ensure you create a classic GH access token with `admin:org` and `user` scopes configured. Set a custom expiration date to one day after the last expected donation date.
![image](https://github.com/thnxdev/wkr-gh-sponsor/assets/72539235/69f248a8-2351-471e-84d5-43eeba9d3f5f)

**Ensure you keep the token stored securely**
Unfortunately, these are the minimum scopes that can create a sponsorship via the GH GraphQL API and they contain write permissions on your account.

