"""
One-shot sync: Google Sheet  ->  enrich (TMDB/OMDB)  ->  MongoDB  ->  back to the Sheet.

Replaces the manual loop of: download CSV, run movieScript.py, drop the Mongo
collection, import CSV, re-upload the CSV to Sheets.

Run from anywhere:
    python3 scripts/syncMovies.py                # full sync
    python3 scripts/syncMovies.py --skip-enrich  # push the sheet to Mongo as-is (no API calls)
    python3 scripts/syncMovies.py --new-only     # enrich only rows with no TMDBId yet, then sync everything
    python3 scripts/syncMovies.py --refresh-omdb # full run, but re-fetch OMDB ratings even for old movies
    python3 scripts/syncMovies.py --check        # test the Sheets and Mongo connections, read-only, then exit
    python3 scripts/syncMovies.py --backup       # dump the Mongo collection and the sheet to scripts/backups/, then exit

Reads settings from the API's .env file (../.env, gitignored), shared with the Go server:
    MONGOURI          = mongodb+srv://...          (same one the API uses)
    MONGO_DB          = jdmovies                   (optional, default jdmovies)
    MONGO_COLLECTION  = movies                     (optional, default movies)
    SHEET_ID          = long id from the Google Sheet URL
    WORKSHEET_NAME    = Masterlist
    GOOGLE_SA_JSON    = /path/to/service-account.json   (share the sheet with its client_email)
    TMDBKEY, TMDBTOKEN
    OMDB_KEY, OMDB_KEY2, OMDB_KEY3                 (extra OMDB keys are optional)

Install once:
    pip3 install -r requirements.txt
"""
import argparse, csv, json, os, sys, time
from datetime import datetime
import requests


# ------------------------------- settings (.env) ----------------------------
def load_env(path):
    """Minimal .env reader: KEY=VALUE lines, optional quotes, # comments. Real env vars win."""
    if os.path.exists(path):
        with open(path, encoding='utf-8') as f:
            for line in f:
                line = line.strip()
                if not line or line.startswith('#') or '=' not in line:
                    continue
                k, v = line.split('=', 1)
                os.environ.setdefault(k.strip(), v.strip().strip('"').strip("'"))


load_env(os.path.join(os.path.dirname(os.path.abspath(__file__)), '..', '.env'))


def env(name, default=None):
    v = os.environ.get(name, default)
    if v is None or v == '':
        raise SystemExit(f'{name} is not set in .env')
    return v


class config:
    mongo_uri = env('MONGOURI')
    mongo_db = env('MONGO_DB', 'jdmovies')
    mongo_collection = env('MONGO_COLLECTION', 'movies')
    sheet_id = env('SHEET_ID')
    worksheet_name = env('WORKSHEET_NAME')
    google_sa_json = env('GOOGLE_SA_JSON')
    tmdbkey = env('TMDBKEY')
    tmdbtoken = env('TMDBTOKEN')
    apikey = env('OMDB_KEY')
    apikey2 = os.environ.get('OMDB_KEY2') or env('OMDB_KEY')
    apikey3 = os.environ.get('OMDB_KEY3') or env('OMDB_KEY')

# ---------------------------------------------------------------------------
# Types written to Mongo. The old Compass CSV import auto-detected these.
# Run `--check` to see the types in your current collection and adjust to match so
# the API keeps working unchanged. Anything not listed is stored as a string.
# ---------------------------------------------------------------------------
CAST = {
    'Year': int,
    'JH_Score': int,
    'Ranking': int,
    'TMDBId': int,
    'Runtime': int,
    'ms_added': int,
    'Dani_Approved': lambda v: str(v).strip().upper() == 'TRUE',
    'Ratings': json.loads,          # '[{"Source": ...}]'  -> list of dicts
    'Provider': json.loads,         # '{"link": ...}'      -> dict
    'Recommendations': json.loads,  # '[581997, 141052]'   -> list of ints
    'Directors': json.loads,        # '["Joel Coen", "Ethan Coen"]' -> list of names
    'Cast': json.loads,             # '["Tom Hanks", "Tim Allen"]'  -> list of names
}
SKIP_EMPTY = True   # Compass omitted empty cells from documents; keep that behavior
MONGO_KEY = 'TMDBId'
BACKUP_DIR = os.path.join(os.path.dirname(os.path.abspath(__file__)), 'backups')   # field used to match sheet rows to Mongo documents

MAX_RETRIES = 5
WAIT_TIME = 2

# OMDB staleness rule: skip the OMDB call when the movie was released more than
# this many days ago AND it already has OMDB data. Ratings for old movies barely
# move, and this keeps a full run well under the 1,000/day free limit.
OMDB_STALE_DAYS = 2 * 365


# ------------------------------- Google Sheets ------------------------------
def open_worksheet():
    import gspread
    gc = gspread.service_account(filename=os.path.expanduser(config.google_sa_json))
    return gc.open_by_key(config.sheet_id).worksheet(config.worksheet_name)


def read_sheet(ws):
    """Return (fieldnames, rows) with every cell as a string, same as csv.DictReader."""
    values = ws.get_all_values()
    fieldnames = values[0]
    rows = [dict(zip(fieldnames, r + [''] * (len(fieldnames) - len(r)))) for r in values[1:]]
    rows = [r for r in rows if r.get('Movie')]   # drop blank trailing rows
    return fieldnames, rows


# Columns written back to the sheet as real numbers / booleans instead of text.
# Everything else is written as literal text so Sheets does not reinterpret
# values like "8.4/10", "97%", "394,884,133" or the JSON columns.
SHEET_NUMERIC = ('Year', 'JH_Score', 'Ranking', 'TMDBId', 'Runtime', 'ms_added')
SHEET_BOOL = ('Dani_Approved',)
RANKING_FORMULA = '=ROW()-1'   # Ranking column is always the row position


def sheet_value(field, v):
    if v == '':
        return ''
    if field in SHEET_NUMERIC:
        try:
            return int(v)
        except ValueError:
            return v
    if field in SHEET_BOOL:
        return str(v).strip().upper() == 'TRUE'
    return v


def write_sheet(ws, fieldnames, rows):
    from gspread.utils import rowcol_to_a1
    values = [fieldnames] + [[sheet_value(f, row.get(f, '')) for f in fieldnames] for row in rows]
    # RAW stops Sheets from turning "8.4/10" into a date or mangling the JSON columns
    ws.update(values=values, range_name='A1', value_input_option='RAW')
    # restore the Ranking formula (RAW would have stored it as text)
    if 'Ranking' in fieldnames and rows:
        c = fieldnames.index('Ranking') + 1
        rng = f'{rowcol_to_a1(2, c)}:{rowcol_to_a1(len(rows) + 1, c)}'
        ws.update(values=[[RANKING_FORMULA]] * len(rows), range_name=rng, value_input_option='USER_ENTERED')
    print(f'Sheet updated: {len(rows)} rows')


# ---------------------------------- Mongo ----------------------------------
def open_collection():
    from pymongo import MongoClient
    client = MongoClient(config.mongo_uri)
    return client[config.mongo_db][config.mongo_collection]


def to_doc(row):
    doc = {}
    for k, v in row.items():
        if SKIP_EMPTY and v == '':
            continue
        if k in CAST and v != '':
            try:
                doc[k] = CAST[k](v)
                continue
            except (ValueError, TypeError):
                pass
        doc[k] = v
    return doc


def write_mongo(col, rows):
    from pymongo import ReplaceOne
    from pymongo.errors import BulkWriteError
    docs = [to_doc(r) for r in rows]
    docs = [d for d in docs if d.get(MONGO_KEY) not in ('', None)]
    if not docs:
        print('Mongo: nothing to write')
        return
    keys = [d[MONGO_KEY] for d in docs]

    # sanity checks against the unique indexes (TMDBId, Ranking) before touching anything
    for field in (MONGO_KEY, 'Ranking'):
        seen = {}
        for d in docs:
            v = d.get(field)
            if v in seen:
                raise SystemExit(f'Mongo: sheet has duplicate {field}={v!r} ({seen[v]!r} and {d["Movie"]!r}); fix the sheet first')
            seen[v] = d['Movie']

    # 1) remove docs for movies no longer in the sheet (what drop+import used to do)
    removed = col.delete_many({MONGO_KEY: {'$nin': keys}}).deleted_count

    # 2) Ranking has a unique index, and rankings shift when a movie is added or moved.
    #    Park every existing Ranking on a value that cannot collide (-TMDBId) so the
    #    replacements below never fight the index.
    col.update_many({}, [{'$set': {'Ranking': {'$multiply': ['$' + MONGO_KEY, -1]}}}])

    # 3) upsert the real documents
    ops = [ReplaceOne({MONGO_KEY: d[MONGO_KEY]}, d, upsert=True) for d in docs]
    try:
        res = col.bulk_write(ops, ordered=False)
    except BulkWriteError as e:
        errs = e.details.get('writeErrors', [])
        print(f'Mongo: {len(errs)} writes FAILED. First few:')
        for err in errs[:5]:
            print(f"  {docs[err['index']]['Movie']!r}: {err.get('errmsg', '')[:200]}")
        raise SystemExit(1)
    print(f'Mongo: {res.upserted_count} inserted, {res.modified_count} updated, {removed} removed, '
          f'{col.estimated_document_count()} total')


def show_types(col):
    doc = col.find_one()
    if not doc:
        print('collection is empty')
        return
    for k, v in doc.items():
        print(f'    {k:20} {type(v).__name__}')


def check_connections():
    """Read-only test of both connections. Nothing is written anywhere."""
    ok = True
    print('Google Sheets ... ', end='', flush=True)
    try:
        ws = open_worksheet()
        fieldnames, rows = read_sheet(ws)
        print(f'OK  sheet "{ws.spreadsheet.title}", tab "{ws.title}", {len(rows)} rows, {len(fieldnames)} columns')
        missing = [c for c in ('Movie', 'Year', MONGO_KEY) if c not in fieldnames]
        if missing:
            ok = False
            print(f'  missing expected columns: {missing}')
    except Exception as e:
        ok = False
        print(f'FAILED\n  {type(e).__name__}: {e}')

    print('MongoDB ......... ', end='', flush=True)
    try:
        col = open_collection()
        col.database.client.admin.command('ping')
        n = col.estimated_document_count()
        print(f'OK  db "{col.database.name}", collection "{col.name}", {n} documents')
        if n == 0:
            print('  note: collection is empty (check mongo_db / mongo_collection names)')
        else:
            print('  field types of one existing document:')
            show_types(col)
    except Exception as e:
        ok = False
        print(f'FAILED\n  {type(e).__name__}: {e}')

    print('\nAll good.' if ok else '\nFix the failures above before running a sync.')
    return ok


def backup():
    """Read-only snapshot of the Mongo collection (JSON, restorable) and the sheet (CSV)."""
    from bson import json_util
    stamp = time.strftime('%Y%m%d-%H%M%S')
    os.makedirs(BACKUP_DIR, exist_ok=True)

    col = open_collection()
    docs = list(col.find())
    mpath = f'{BACKUP_DIR}/mongo-{col.database.name}.{col.name}-{stamp}.json'
    with open(mpath, 'w', encoding='utf-8') as f:
        f.write(json_util.dumps(docs, indent=1))
    print(f'Mongo: {len(docs)} documents -> {mpath}')

    ws = open_worksheet()
    fieldnames, rows = read_sheet(ws)
    spath = f'{BACKUP_DIR}/sheet-{ws.title}-{stamp}.csv'
    write_csv(fieldnames, rows, spath)
    print('Restore Mongo with:  python3 syncMovies.py --restore', mpath)


def restore(path):
    """Replace the whole Mongo collection with a backup JSON file."""
    from bson import json_util
    with open(path, encoding='utf-8') as f:
        docs = json_util.loads(f.read())
    col = open_collection()
    answer = input(f'Replace ALL {col.estimated_document_count()} docs in {col.database.name}.{col.name} '
                   f'with {len(docs)} docs from {path}? [y/N] ')
    if answer.strip().lower() != 'y':
        print('aborted'); return
    col.delete_many({})
    col.insert_many(docs)
    print(f'restored {len(docs)} documents')


# -------------------------------- Enrichment -------------------------------
def get_rating(ratings, source):
    match = [r['Value'] for r in ratings if r['Source'] == source]
    return match[0] if match else 'N/A'


def has_omdb_data(row):
    return bool(row.get('Ratings')) and bool(row.get('Rated'))


def needs_omdb(row, release_date, year, refresh=False):
    """True if the OMDB call should be made for this row."""
    if refresh or not has_omdb_data(row):
        return True
    try:
        released = datetime.strptime(release_date, '%Y-%m-%d')
    except (TypeError, ValueError):
        released = datetime(year, 1, 1)
    age_days = (datetime.now() - released).days
    return age_days <= OMDB_STALE_DAYS


def enrich(rows, new_only=False, refresh_omdb=False):
    """Same logic as movieScript.py. Mutates rows in place. Returns False if OMDB quota ran out."""
    omdb_keys = list(dict.fromkeys([config.apikey, config.apikey2, config.apikey3]))
    omdb_key_index = 0
    currentTime = round(time.time() * 1000)
    omdb_calls = omdb_skipped = 0

    for index, row in enumerate(rows):
        if new_only and row['TMDBId']:
            continue

        title = row['Movie']
        year = int(row['Year'])

        if not row['TMDBId']:
            url = 'https://api.themoviedb.org/3/search/movie'
            params = {'api_key': config.tmdbkey, 'query': title, 'year': year}
            search = requests.get(url, params=params).json()
            print(f'{url}?query={title}&year={year}')
            path = search['results'][0]['poster_path']
            tmdbcode = int(search["results"][0]["id"])
            row['Poster'] = f'https://image.tmdb.org/t/p/w500{path}'
            row['TMDBId'] = str(tmdbcode)
            row['ms_added'] = str(currentTime)
        else:
            tmdbcode = int(row['TMDBId'])

        tmdb_url = f'https://api.themoviedb.org/3/movie/{tmdbcode}'

        if not row['Actors']:
            castAndCrew = requests.get(f'{tmdb_url}/credits?api_key={config.tmdbkey}').json()
            actorString = ', '.join(actor["name"] for actor in castAndCrew["cast"])
            print(title, actorString)
            row['Actors'] = actorString if actorString else "N/A"
            row['Director'] = ', '.join(c["name"] for c in castAndCrew["crew"] if c["job"] == "Director")

        movieInfo = requests.get(f'{tmdb_url}?api_key={config.tmdbkey}').json()
        boxofficeTotal = movieInfo.get('revenue', 'N/A')
        try:
            country = movieInfo['origin_country'][0]
        except (KeyError, IndexError):
            country = 'US'
        imdbid = movieInfo["imdb_id"]
        row['Plot'] = movieInfo["overview"]
        row['BoxOffice'] = f"{boxofficeTotal:,}" if isinstance(boxofficeTotal, int) else boxofficeTotal
        row['Budget'] = f"{movieInfo['budget']:,}"
        row['Runtime'] = f"{movieInfo['runtime']:,}"
        row['origin_country'] = country

        providers = requests.get(f'{tmdb_url}/watch/providers?api_key={config.tmdbkey}').json()
        if providers['results'] and 'CA' in providers['results']:
            row['Provider'] = json.dumps(providers['results']['CA'])
        else:
            row['Provider'] = "{}"

        recoUrl = f'{tmdb_url}/recommendations?api_key={config.tmdbkey}'
        for i in range(0, MAX_RETRIES):
            try:
                recommendations = requests.get(recoUrl).json()
                row['Recommendations'] = str([item['id'] for item in recommendations['results']])
                break
            except Exception:
                print(f'FAILED {i}, {recoUrl}, {title}, {year}')
                if i < MAX_RETRIES - 1:
                    time.sleep(WAIT_TIME)

        videos = requests.get(f'{tmdb_url}/videos?api_key={config.tmdbkey}').json()
        trailerList = [r for r in videos['results'] if r['type'] == 'Trailer']
        if trailerList:
            selected_trailer = next((t for t in trailerList if t['official']), trailerList[0])
        elif videos['results']:
            selected_trailer = videos['results'][0]
        else:
            selected_trailer = ''
        row['Trailer'] = f'https://www.youtube.com/embed/{selected_trailer["key"]}' if selected_trailer != '' else ''

        if row['JH_Score']:
            value = round(float(row['JH_Score']) / 5) / 2
            if value == 0.0:
                value = 0.5
            headers = {'Content-Type': 'application/json;charset=utf8', 'Authorization': f'Bearer {config.tmdbtoken}'}
            requests.post(f'{tmdb_url}/rating', headers=headers, json={"value": value})

        # OMDB SECTION
        if not needs_omdb(row, movieInfo.get('release_date'), year, refresh_omdb):
            omdb_skipped += 1
            print(title, year, index, '(OMDB skipped: older than 2 years and already filled)')
            continue
        omdb_calls += 1
        omdb = {}
        while omdb_key_index < len(omdb_keys):
            omdb = requests.get(f'http://www.omdbapi.com/?apikey={omdb_keys[omdb_key_index]}&i={imdbid}&type=movie').json()
            if omdb.get("Response") == "False" and 'limit' in omdb.get("Error", "").lower():
                omdb_key_index += 1
                if omdb_key_index < len(omdb_keys):
                    print(f'OMDB key hit its daily limit, switching to key {omdb_key_index + 1} of {len(omdb_keys)}')
                continue
            break
        if omdb_key_index >= len(omdb_keys):
            print('All OMDB keys hit their daily limit, stopping so progress is saved. Re-run later to finish.')
            print(f'OMDB calls made: {omdb_calls}, skipped by staleness rule: {omdb_skipped}')
            return False
        if omdb.get("Response") == "False" or "Ratings" not in omdb:
            error = omdb.get("Error", "unknown error")
            print(f'OMDB FAILED: {error}, {title}, {year}, {index}')
        else:
            ratings = omdb["Ratings"]
            row['RottenTomatoes'] = get_rating(ratings, 'Rotten Tomatoes')
            row['IMDB'] = get_rating(ratings, 'Internet Movie Database')
            row['Metacritic'] = get_rating(ratings, 'Metacritic')
            row['Ratings'] = json.dumps(ratings)
            row['Rated'] = omdb.get("Rated", "N/A")

        print(title, year, index)
    print(f'OMDB calls made: {omdb_calls}, skipped by staleness rule: {omdb_skipped}')
    return True


def derive_name_lists(fieldnames, rows):
    """Directors/Cast (JSON arrays) are always derived from the Director/Actors
    strings, so the columns can never drift apart. Adds the columns if missing."""
    for src, dst in (('Director', 'Directors'), ('Actors', 'Cast')):
        if dst not in fieldnames:
            fieldnames.insert(fieldnames.index(src) + 1, dst)
        for row in rows:
            value = row.get(src, '')
            if value.strip().upper() == 'N/A':
                value = ''
            names = [n.strip() for n in value.split(',') if n.strip()]
            row[dst] = json.dumps(names) if names else ''


# ----------------------------------- CSV -----------------------------------
def write_csv(fieldnames, rows, path):
    with open(path, 'w', newline='', encoding='utf-8') as f:
        writer = csv.DictWriter(f, fieldnames=fieldnames)
        writer.writeheader()
        writer.writerows(rows)
    print(f'CSV written: {path}')


# ----------------------------------- main ----------------------------------
def main():
    ap = argparse.ArgumentParser()
    ap.add_argument('--skip-enrich', action='store_true', help='no TMDB/OMDB calls, just sheet -> Mongo')
    ap.add_argument('--new-only', action='store_true', help='enrich only rows with no TMDBId, then sync all')
    ap.add_argument('--refresh-omdb', action='store_true', help='ignore the staleness rule and re-fetch OMDB for every row')
    ap.add_argument('--check', action='store_true', help='test Sheets and Mongo connections read-only, then exit')
    ap.add_argument('--backup', action='store_true', help='dump Mongo collection + sheet to scripts/backups/, then exit')
    ap.add_argument('--restore', metavar='FILE', help='replace the Mongo collection with a backup JSON file')
    args = ap.parse_args()

    if args.backup:
        backup(); return
    if args.restore:
        restore(args.restore); return

    if args.check:
        sys.exit(0 if check_connections() else 1)

    ws = open_worksheet()
    fieldnames, rows = read_sheet(ws)
    print(f'Read {len(rows)} rows from sheet "{config.worksheet_name}"')

    try:
        if not args.skip_enrich:
            enrich(rows, new_only=args.new_only, refresh_omdb=args.refresh_omdb)
    finally:
        # always persist whatever progress was made, even if a row crashed
        derive_name_lists(fieldnames, rows)
        write_sheet(ws, fieldnames, rows)
        write_mongo(open_collection(), rows)


if __name__ == '__main__':
    main()
