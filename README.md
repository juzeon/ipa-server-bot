# IPA Server Bot

This is a telegram bot that generates a link to install signed .ipa files directly on iOS devices.

[Demo](https://t.me/IPAServerBot)

The bot talks to Telegram over MTProto with [gotd/botapi](https://github.com/gotd/botapi), so it can download `.ipa` files larger than the 20MB cloud Bot API limit. Each upload is written to a temporary file and deleted after processing, including when parsing or upload fails.

## Configure

1. Create a bot using [@BotFather](https://t.me/botfather) and get your bot token.
2. Create an application at [my.telegram.org](https://my.telegram.org) and copy its `api_id` and `api_hash`.
3. If this bot has already been used with `https://api.telegram.org`, log it out once: `https://api.telegram.org/bot<your bot token>/logOut`.
4. Copy `config.yml.example` to `config.yml` and fill it in. Pass another location with `-c`:

```bash
./ipa-server-bot -c /etc/ipa-server-bot/config.yml
```

Relative `session_file` paths are relative to the process working directory.

## Run

```bash
go build -o ipa-server-bot
./ipa-server-bot
```

## systemd

Install the binary, a dedicated user, and the unit, then enable it:

```bash
go build -o ipa-server-bot
sudo install -m 0755 ipa-server-bot /usr/local/bin/ipa-server-bot
sudo useradd --system --home-dir /var/lib/ipa-server-bot --create-home ipa-server-bot
sudo install -d -m 0700 -o ipa-server-bot -g ipa-server-bot /etc/ipa-server-bot
sudo install -m 0600 -o ipa-server-bot -g ipa-server-bot config.yml /etc/ipa-server-bot/config.yml
sudo install -m 0644 ipa-server-bot.service /etc/systemd/system/ipa-server-bot.service
sudo systemctl daemon-reload
sudo systemctl enable --now ipa-server-bot
```

The unit runs as `ipa-server-bot` with working directory `/var/lib/ipa-server-bot`, so the MTProto session and `sessions/` stay there. Put absolute paths in that config if you do not want them resolved from the working directory.

## List of commands

```text
start - Start the bot
list - List my applications
```
