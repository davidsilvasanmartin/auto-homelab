import shutil
import subprocess
import sys
import time
from pathlib import Path

###### TODO REVIEW !!!!!!

WARNING = """
NOTE: For some reason Docker may change permissions of some directories, so it is possible
that this operation fails due to that. If this happens, set yourself manually (chown) as the
owner of the problematic files.

WARNING: This operation will DELETE Immich containers, volumes, and all existing data.
Type 'yes' to confirm and continue, or anything else to abort.
""".strip()


def eprintln(msg: str) -> None:
    print(msg, file=sys.stderr)


def read_env_file(path: Path) -> dict[str, str]:
    env: dict[str, str] = {}
    if not path.exists():
        raise FileNotFoundError(f"{path} not found. Please create it before running this command.")
    for line in path.read_text(encoding="utf-8").splitlines():
        s = line.strip()
        if not s or s.startswith("#"):
            continue
        # Support KEY=VAL and KEY="VAL with = inside"
        if "=" not in s:
            continue
        key, val = s.split("=", 1)
        key = key.strip()
        val = val.strip()
        if val.startswith(("'", '"')) and val.endswith(("'", '"')) and len(val) >= 2:
            val = val[1:-1]
        env[key] = val
    return env


def confirm_or_exit() -> None:
    eprintln(WARNING)
    try:
        resp = input("Confirm (type 'yes'): ").strip()
    except EOFError:
        eprintln("Aborted: no confirmation received.")
        sys.exit(1)
    if resp != "yes":
        eprintln("Aborted by user.")
        sys.exit(1)


def prompt_for_paths() -> tuple[Path, Path]:
    try:
        db_path = Path(input("Enter path to your restored Immich database .sql file: ").strip()).expanduser()
        if not (db_path.is_file() and db_path.suffix == ".sql"):
            eprintln(f"Error: Database dump must be an existing .sql file: {db_path}")
            sys.exit(1)

        upload_dir = Path(
            input('Enter path to your restored Immich "upload" directory: ').strip()
        ).expanduser()
        if not upload_dir.is_dir():
            eprintln(f"Error: Invalid path: {upload_dir}")
            sys.exit(1)
        return db_path, upload_dir
    except EOFError:
        eprintln("Aborted: no input received.")
        sys.exit(1)


def ensure_dir(path: Path) -> None:
    path.mkdir(parents=True, exist_ok=True)


def clear_directory_contents(dir_path: Path) -> None:
    if not dir_path.exists():
        return
    if not dir_path.is_dir():
        raise NotADirectoryError(f"Expected a directory, got: {dir_path}")
    for child in dir_path.iterdir():
        try:
            if child.is_symlink() or child.is_file():
                child.unlink(missing_ok=True)
            elif child.is_dir():
                shutil.rmtree(child)
        except Exception as exc:
            raise RuntimeError(f"Failed to remove {child}: {exc}") from exc


def copy_directory_contents(src_dir: Path, dst_dir: Path) -> None:
    ensure_dir(dst_dir)
    # Python 3.13: copytree supports dirs_exist_ok
    for item in src_dir.iterdir():
        src = item
        dst = dst_dir / item.name
        if src.is_dir():
            shutil.copytree(src, dst, dirs_exist_ok=True)
        else:
            shutil.copy2(src, dst)


def run(
    cmd: list[str] | str, check: bool = True, input_bytes: bytes | None = None
) -> subprocess.CompletedProcess:
    shell = isinstance(cmd, str)
    return subprocess.run(
        cmd,
        shell=shell,
        check=check,
        input=input_bytes,
        stdout=sys.stdout,
        stderr=sys.stderr,
    )


def main() -> int:
    # 1) Confirm
    confirm_or_exit()

    # 2) Load .env
    try:
        env = read_env_file(Path(".env"))
    except Exception as exc:
        eprintln(f"Error: {exc}")
        return 1

    # Required env vars
    required_keys = [
        "HOMELAB_IMMICH_WEB_UPLOAD_PATH",
        "HOMELAB_IMMICH_DB_DATA_PATH",
        "HOMELAB_IMMICH_ML_CACHE_DATA_PATH",
        "HOMELAB_IMMICH_REDIS_DATA_PATH",
        "HOMELAB_IMMICH_DB_CONTAINER_NAME",
        "HOMELAB_IMMICH_DB_PASSWORD",
        "HOMELAB_IMMICH_DB_USER",
        "HOMELAB_IMMICH_DB_DATABASE",
    ]
    missing = [k for k in required_keys if not env.get(k)]
    if missing:
        eprintln(f"Error: Missing required .env variables: {', '.join(missing)}")
        return 1

    db_container = env["HOMELAB_IMMICH_DB_CONTAINER_NAME"]
    db_password = env["HOMELAB_IMMICH_DB_PASSWORD"]
    db_user = env["HOMELAB_IMMICH_DB_USER"]
    db_name = env["HOMELAB_IMMICH_DB_DATABASE"]

    upload_path = Path(env["HOMELAB_IMMICH_WEB_UPLOAD_PATH"]).expanduser()
    db_data_path = Path(env["HOMELAB_IMMICH_DB_DATA_PATH"]).expanduser()
    ml_cache_path = Path(env["HOMELAB_IMMICH_ML_CACHE_DATA_PATH"]).expanduser()
    redis_path = Path(env["HOMELAB_IMMICH_REDIS_DATA_PATH"]).expanduser()

    # 3) Prompt for restored sources
    restored_db, restored_upload = prompt_for_paths()

    # 4) Stop and drop containers/volumes
    try:
        run(
            [
                "docker",
                "compose",
                "down",
                "-v",
                "immich-redis",
                "immich-machine-learning",
                "immich-db",
                "immich",
            ]
        )
    except subprocess.CalledProcessError:
        eprintln("Warning: 'docker compose down' failed. Continuing, but this may cause issues.")

    # 5) Clear data directories
    try:
        for p in (upload_path, db_data_path, ml_cache_path, redis_path):
            ensure_dir(p)
            clear_directory_contents(p)
    except Exception as exc:
        eprintln(str(exc))
        return 1

    # 6) Restore files
    try:
        ensure_dir(upload_path)
        copy_directory_contents(restored_upload, upload_path)
    except Exception as exc:
        eprintln(f"Error copying upload directory: {exc}")
        return 1

    # 7) Start DB-related services
    try:
        run(["docker", "compose", "up", "-d", "immich-redis", "immich-machine-learning", "immich-db"])
    except subprocess.CalledProcessError as exc:
        eprintln(f"Error starting services: {exc}")
        return 1

    # 8) Wait for DB
    time.sleep(30)

    # 9) Restore database
    try:
        # Pipe SQL into docker exec -i <container> /bin/bash -c "PGPASSWORD=... psql ..."
        # TODO this is breaking
        # psql_cmd = (
        #     f"/bin/bash -c {shlex.quote(f\"PGPASSWORD='{db_password}' "
        #     f"psql --username='{db_user}' --dbname='{db_name}'\")}"
        # )
        psql_cmd = "/bin/bash -c whatever"
        with restored_db.open("rb") as f:
            run(["docker", "exec", "-i", db_container, psql_cmd], input_bytes=f.read())
    except subprocess.CalledProcessError as exc:
        eprintln(f"Error restoring database: {exc}")
        return 1
    except FileNotFoundError:
        eprintln(f"Error: SQL file not found: {restored_db}")
        return 1

    # 10) Start Immich
    try:
        run(["docker", "compose", "up", "-d", "immich"])
    except subprocess.CalledProcessError as exc:
        eprintln(f"Error starting Immich: {exc}")
        return 1

    print("Immich restoration completed successfully.")
    return 0


if __name__ == "__main__":
    sys.exit(main())
