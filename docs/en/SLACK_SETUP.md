# scat Slack Setup Guide

To use `scat` with the Slack provider, you need to create a Slack App and get a Bot Token with the appropriate permissions (scopes). This guide will walk you through the process.

---

### Step 1: Create a Slack App

1.  Go to the [Slack API site](https://api.slack.com/apps) and log in to your account.

2.  Click the **"Create New App"** button.

3.  In the dialog that appears, select **"From scratch"**.

4.  Enter an app name (e.g., `My Scat Bot`), select the workspace you want to install it to, and click **"Create App"**.

**Note:** The app icon and description suggestions provided below are optional. You are free to use your own preferred icon and descriptions. This documentation serves as a reference if you need guidance.

### Display Information

Please set the following descriptions in the "Display Information" section of your Slack app.

#### Short description

CLI tool to streamline Slack operations: post messages, export logs, manage channels.

#### Long description

Scat is a powerful Command Line Interface (CLI) tool designed to enhance your interaction with Slack. It enables users to efficiently perform various Slack operations directly from their terminal, including posting messages, retrieving channel lists, exporting logs, uploading files, and managing profiles. Built for developers and power users, Scat helps automate tasks through scripting and simplifies daily workflows.

---

### App Icon

You can use the provided `scat_icon.png` located in the `docs/` directory as the app icon for your Slack application.

---

### Step 2: Add Permissions (Scopes)

Once you are on the app's management screen, you need to add the permissions that `scat` requires.

1.  From the left sidebar, select **"OAuth & Permissions"**.

2.  Scroll down to the **"Scopes"** section.

3.  Under **"Bot Token Scopes"**, click the **"Add an OAuth Scope"** button and add each of the following scopes:

    **Core Scopes (for posting):**
    *   `chat:write`: Required to post messages to public channels.
    *   `files:write`: Required to upload files.
    *   `channels:join`: Required for the bot to automatically join public channels before posting.

    **DM & User Scopes (for Direct Messages and user info):**
    *   `im:write`: Required to open a direct message channel with a user.
    *   `users.read`: Required to find a user by their @mention name (for DMs) and to resolve user IDs to names (for log exports).

    **Optional Scopes (for extra features):**
    *   `channels:manage`: Required for the `channel create` command to create public channels.
    *   `groups:write`: Required for the `channel create` command to create private channels.
    *   `channels:read`: Required for the `channel list` command.
    *   `groups:read`: Required for the `channel list` command to see private channels.
    *   `chat:write.customize`: Required if you want to override the bot's name or icon using the `--username` or `--iconemoji` flags.
    *   `usergroups:read`: Required to resolve user group names for invitations.

    **Export Scopes (for `export log` command):**
    *   `channels:history`: Required to read message history from public channels.
    *   `groups:history`: Required to read message history from private channels.
    *   `files:read`: Required to download attached files.

### Step 3: Install the App to Your Workspace

After adding the scopes, you can install the app to your workspace to generate a token.

1.  Scroll back to the top of the "OAuth & Permissions" page.

2.  Click the **"Install to Workspace"** button.

3.  On the next screen, click **"Allow"** to authorize the app.

### Step 4: Get Your Bot Token

After installation, the page will refresh, and you will see a **"Bot User OAuth Token"**.

*   Copy this token, which starts with `xoxb-`. This is the token you will use to configure `scat`.

### Step 5: Configure scat

Use the token you just copied to create or update a `scat` profile.

```bash
# Create a new profile named "my-slack"
scat profile add my-slack-workspace --provider slack --channel "#general"

# After running the command above, you will be prompted to enter your token.
# Paste the "xoxb-..." token you copied in Step 4 and press Enter.
Enter Token (will not be displayed): [paste your token here]
```

Your setup is now complete.

### Step 6: Invite the Bot to Channels

For the bot to be able to post in a channel (especially private channels), it must be a member of that channel.

*   In each Slack channel you want to post to, invite the bot using the following command:

    ```
    /invite @<your-app-name>
    ```

You are now ready to post to Slack using `scat`!
