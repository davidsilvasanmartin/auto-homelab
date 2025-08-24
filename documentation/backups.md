# Backups

Examples of running cloud backup with the `cloud_backup.py` script:

``` bash
   # Run a full backup
   python cloud_backup.py --command backup
   
   # Run specific commands
   python cloud_backup.py --command init    # Initialize repository
   python cloud_backup.py --command check   # Check repository integrity
   python cloud_backup.py --command list    # List all snapshots
   python cloud_backup.py --command prune   # Prune old backups
   # Restore the full backup into a local directory
   python cloud_backup.py --command restore --restore-dir ./restore
```

# How Restic and Backblaze B2 Backups Work

Let me explain what's happening in the `cloud-backup.py` script and how the backup process works with restic and
Backblaze B2:

## Overview of the Backup Process

1. **What the script does**: The script is a wrapper around the `restic` command-line tool, which handles the actual
   backup operations. It provides a Python interface to manage the backup process.

2. **Where the backups are stored**: Backups are stored directly in Backblaze B2 cloud storage. There is no local copy
   kept by default (beyond your original files).

3. **How Backblaze connection works**: The connection to Backblaze B2 is handled entirely by restic itself, not by the
   Python script.

## Connection to Backblaze B2

The reason you don't see explicit code for the Backblaze connection is because:

1. **Restic handles the connection**: Restic has built-in support for multiple cloud storage providers, including
   Backblaze B2.

2. **Authentication via environment variables**: The script sets these environment variables that restic uses:

```python
self.env['RESTIC_REPOSITORY'] = self.repository
self.env['B2_ACCOUNT_ID'] = self.b2_account_id
self.env['B2_ACCOUNT_KEY'] = self.b2_account_key
self.env['RESTIC_PASSWORD'] = self.restic_password
```

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

## To Restore Files

If you need to restore files from Backblaze, you would use restic commands like:

```shell script
# Restore the latest snapshot to a specific directory
restic restore latest --target /path/to/restore/directory

# List snapshots
restic snapshots

# Restore a specific snapshot
restic restore <snapshot-id> --target /path/to/restore/directory
```

The script could be extended to include restore functionality as well.

## Local Cache

Restic does maintain a small local cache to speed up operations, usually in `~/.cache/restic/`, but this doesn't store
your backup data - just metadata to make operations faster.

## To Keep a Local Copy Too

If you want to maintain both local and cloud backups, you would need to:

1. First back up to a local repository
2. Then back up to Backblaze B2

This would require modifications to the script or a separate workflow.
