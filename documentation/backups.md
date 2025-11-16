# Backups

## Cloud Backups

Examples of running cloud backup with the Go application:

``` bash
   # Run a full backup (init, backup, prune)
   go run . backup cloud

   # Run specific commands
   go run . backup cloud init              # Initialize repository
   go run . backup cloud check             # Check repository integrity
   go run . backup cloud list              # List all snapshots
   go run . backup cloud prune             # Prune old backups
   go run . backup cloud restore ./restore # Restore to a local directory
   go run . backup cloud ls-files <snapshot-id>  # List files in a snapshot
```

# How Restic and Backblaze B2 Backups Work

Let me explain what's happening in the cloud backup implementation and how the backup process works with restic and
Backblaze B2:

## Overview of the Backup Process

1. **What the application does**: The Go application is a wrapper around the `restic` command-line tool, which handles the actual
   backup operations. It provides a clean interface to manage the backup process.

2. **Where the backups are stored**: Backups are stored directly in Backblaze B2 cloud storage. There is no local copy
   kept by default (beyond your original files).

3. **How Backblaze connection works**: The connection to Backblaze B2 is handled entirely by restic itself, not by the
   Go application.

## Connection to Backblaze B2

The reason you don't see explicit code for the Backblaze connection is because:

1. **Restic handles the connection**: Restic has built-in support for multiple cloud storage providers, including
   Backblaze B2.

2. **Authentication via environment variables**: The application sets these environment variables that restic uses:
   - `RESTIC_REPOSITORY`: The repository URL (e.g., `b2:bucket-name:path/prefix`)
   - `B2_ACCOUNT_ID`: Your Backblaze B2 key ID
   - `B2_ACCOUNT_KEY`: Your Backblaze B2 application key
   - `RESTIC_PASSWORD`: The encryption password for your restic repository

3. **Repository URL format**: The `RESTIC_REPOSITORY` variable uses a special URL format that tells restic which backend
   to use. For Backblaze B2, it's:

```
b2:bucket-name:optional/path/prefix/
```

When you run restic commands with these environment variables set, restic automatically:

1. Authenticates with Backblaze using the account ID and key
2. Connects to the specified bucket
3. Stores or retrieves data in that location

## Storage Details

1. **Remote storage only**: By default, this setup only stores your backups in Backblaze B2, not locally. The backup
   path (`./backup` by default) is the source of files to backup, not where backups are stored.

2. **Deduplication**: Restic uses deduplication, meaning it only uploads unique blocks of data. If you back up again and
   most files haven't changed, it will only upload the changes.

3. **Encryption**: All data is encrypted before upload using the password you set in `RESTIC_PASSWORD`.

## Repository Structure

Restic creates several special files in your B2 bucket:

1. **Repository structure**: When you initialize a repository, restic creates a structure to track snapshots, data
   blocks, and metadata.

2. **Snapshots**: Each time you run a backup, restic creates a new snapshot referencing the data blocks for that backup.

## Local Cache

Restic does maintain a small local cache to speed up operations, usually in `~/.cache/restic/`, but this doesn't store
your backup data - just metadata to make operations faster.

## To Keep a Local Copy Too

If you want to maintain both local and cloud backups, you would need to:

1. First back up to a local repository
2. Then back up to Backblaze B2

This would require modifications to the script or a separate workflow.
