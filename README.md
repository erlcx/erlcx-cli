# ERLCX CLI

`erlcx` is the command-line tool for ERLCX.

For now, it helps ER:LC designers upload vehicle liveries to Roblox in bulk and generate an `IDs.txt` file automatically.

More ERLCX marketplace features may be added to this CLI later.

## What It Does

- Uploads multiple livery images to Roblox.
- Gets the Roblox image/texture ID for each uploaded image.
- Creates or updates an `IDs.txt` file.
- Skips images that did not change since the last upload.
- Can skip raw template images when you provide a templates folder.
- Works from a simple command window.

## What It Does Not Do

- It does not ask for your Roblox password.
- It does not control your browser or scrape Roblox pages.
- It does not delete Roblox assets.
- It does not edit your images.

## First Time Setup

Log in with Roblox:

```powershell
erlcx auth login
```

Choose `Custom Roblox OAuth app` and paste your app's client ID and client secret when asked.

Your browser will open a Roblox login/permission page. After you approve it, return to the command window.

Check who is logged in:

```powershell
erlcx auth status
```

Log out when needed:

```powershell
erlcx auth logout
```

To remove the saved custom OAuth app from this PC:

```powershell
erlcx auth clear-oauth-app
```

## Preview Before Uploading

Before uploading, run a scan:

```powershell
erlcx scan "D:\Designs\Sheriff Pack"
```

This shows what the tool would upload, skip, or reject.

## Upload A Pack

Upload all new or changed images:

```powershell
erlcx upload "D:\Designs\Sheriff Pack"
```

The tool will:

1. Scan the pack.
2. Skip unchanged images.
3. Upload new or changed images.
4. Save the Roblox image/texture IDs.
5. Write `IDs.txt`.

## Dry Run

Use dry run when you want to see what would happen without uploading anything:

```powershell
erlcx upload "D:\Designs\Sheriff Pack" --dry-run
```

## Using A Templates Folder

If your pack contains raw ER:LC templates next to finished designs, you can provide a templates folder:

```powershell
erlcx upload "D:\Designs\Sheriff Pack" --templates "D:\ERLC Templates"
```

The tool compares images in your pack against images in the templates folder. If an image is exactly the same as a template image, it is skipped.

## Uploading To A Group

To upload assets under a Roblox group, include the group ID:

```powershell
erlcx upload "D:\Designs\Sheriff Pack" --creator group --group-id 123456
```

You must be logged in to a Roblox account that has permission to upload assets to that group.

## Generated Files

### `IDs.txt`

This is the file you use after uploading.

Example:

```txt
Falcon Stallion 350 2015
Back: 1234567890
Front: 1234567891
Left: 1234567892
Right: 1234567893
Top: 1234567894
```

### `.erlcx-upload.lock.json`

This file helps the tool remember what it already uploaded.

Do not edit it unless you know what you are doing. It does not contain your Roblox password, cookie, or login token.

If you delete it, the tool will no longer know which images were already uploaded and may upload everything again.

## Regenerate `IDs.txt`

If the lock file already exists and you only need to recreate `IDs.txt`, run:

```powershell
erlcx ids "D:\Designs\Sheriff Pack"
```

## Clean Old Lock Entries

If you deleted images from your pack and want to remove old entries from the lock file:

```powershell
erlcx lock clean "D:\Designs\Sheriff Pack"
```

This only cleans the local lock file. It does not delete anything from Roblox.

## Optional Config File

You can create a `.erlcx-uploader.json` file inside your pack folder to save common settings.

Example:

```json
{
  "templatesDir": "D:\\ERLC Templates",
  "outputFile": "IDs.txt",
  "skipNamePatterns": [
    "*_raw.png",
    "*_reference.png"
  ]
}
```

## License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.
