# IPA Server Bot

This is a telegram bot that generates a link to install signed .ipa files directly on iOS devices.

## Deployment

1. Launch a telegram-bot-api instance:

```bash
docker run -d -p 8081:8081 --name=telegram-bot-api --restart=always -v ./data:/var/lib/telegram-bot-api -e TELEGRAM_API_ID=601761 -e TELEGRAM_API_HASH=20a3432aab43f24bb4460fceac5ba38d -e TELEGRAM_LOCAL=1 aiogram/telegram-bot-api:latest
```

You need to use a self-hosted telegram-bot-api service because it can handle files that are larger than 20MB.

Launch an http-server on `./data/data`:

```bash
# install http-server via npm if you don't have it: npm i -g http-server
http-server -p 52138
```

Map paths on your nginx. Here is an example configuration:

```nginx
location /file/bot {
    proxy_pass http://127.0.0.1:52138/;
}

location / {
    proxy_pass http://127.0.0.1:8081;
}
```

2. Create a bot using [@BotFather](https://t.me/botfather) and get your bot token.
3. Log out your bot from the official bot api service. Simply access `https://api.telegram.org/bot<your bot token>/logOut`.
4. Copy `config.yml.example` to `config.yml` and fill in your configuration details.
5. Build and run the bot:

```bash
go build
chmod +x ipa-bot-server
./ipa-bot-server
```