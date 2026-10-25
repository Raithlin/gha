"""Publish bounded master history around unmodified, pinned Groma exports."""

import argparse
import html
import json
from pathlib import Path
import subprocess


def git(repo, *args):
    return subprocess.check_output(['git', '-C', str(repo), *args], text=True).strip()


def has_architecture(repo, revision):
    return subprocess.run(['git', '-C', str(repo), 'cat-file', '-e',
                           f'{revision}:.groma/project.md'],
                          stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL).returncode == 0


def project_metadata(directory):
    # This envelope belongs to the pinned Groma export format. Fail explicitly
    # when it changes rather than silently publishing missing descriptions.
    script = (directory / 'snapshot.js').read_text()
    prefix = 'globalThis.dispatchEvent(new CustomEvent("groma:published", { detail: '
    suffix = ' }));'
    if not script.startswith(prefix) or not script.rstrip().endswith(suffix):
        raise ValueError('unsupported Groma snapshot envelope; check the pinned version')
    data = json.loads(script[len(prefix):].rstrip()[:-len(suffix)])
    return {key: data['project'].get(key, '') for key in ('title', 'description', 'overview')}


def build(repo, output, url, *, revision='HEAD', limit=30, groma='groma'):
    if limit < 1:
        raise ValueError('history limit must be positive')
    head = git(repo, 'rev-parse', '--verify', f'{revision}^{{commit}}')
    if not has_architecture(repo, head):
        raise ValueError('no committed Groma architecture at the selected revision')
    if output.exists():
        raise ValueError('output directory already exists; choose a new directory')
    commits = git(repo, 'rev-list', '--first-parent', f'--max-count={limit}', head).splitlines()
    revisions = []
    for commit in commits:
        if not has_architecture(repo, commit):
            continue
        date, subject = git(repo, 'show', '-s', '--format=%cI%x00%s', commit).split('\0', 1)
        directory = output / 'revisions' / commit
        subprocess.run([groma, 'export', str(directory), '--revision', commit,
                        '--url', f'{url.rstrip("/")}/revisions/{commit}/'],
                       cwd=repo, check=True)
        revisions.append({'id': commit, 'date': date, 'subject': subject,
                          **project_metadata(directory)})
    (output / 'index.html').write_text(render(revisions, limit))
    return revisions


def render(revisions, limit):
    latest = revisions[0]
    options = '\n'.join(
        f'<option value="{item["id"]}">{html.escape(item["id"][:7] + " · " + item["subject"])}</option>'
        for item in revisions)
    # Escaping '<' also prevents commit messages from closing the JSON script.
    data = json.dumps(revisions).replace('<', '\\u003c').replace('>', '\\u003e').replace('&', '\\u0026')
    page = '''<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>GHA architecture · master</title>
  <style>
    *{box-sizing:border-box} html,body{margin:0;height:100%}
    body{display:flex;flex-direction:column;background:#101416;color:#edf2f0;font:15px/1.5 system-ui,sans-serif}
    header{padding:12px 20px;border-bottom:1px solid #3b4944;flex:none}
    .toolbar{display:flex;align-items:center;gap:12px;flex-wrap:wrap}
    h1{font-size:18px;margin:0 auto 0 0} label{font-weight:600}
    select{max-width:100%;width:440px;padding:6px;background:#18211e;color:inherit;border:1px solid #63776d;border-radius:4px}
    a{color:#9cddbf} p{margin:6px 0} .note{font-size:12px;color:#b8c6bf}
    summary{cursor:pointer;color:#9cddbf} details{margin-top:8px}
    #overview{white-space:pre-wrap;overflow-wrap:anywhere;max-height:40vh;overflow:auto;padding:10px 0}
    iframe{width:100%;flex:1;min-height:0;border:0;background:#101416}
    @media(max-width:640px){header{padding:10px}h1{width:100%}select{width:100%}.toolbar{gap:6px}}
  </style>
</head>
<body>
  <header>
    <div class="toolbar">
      <h1 id="project-title">__TITLE__</h1>
      <label for="revision">master history</label>
      <select id="revision">__OPTIONS__</select>
      <a id="commit-link" href="https://github.com/Raithlin/gha/commit/__HEAD__">View commit</a>
    </div>
    <p id="description">__DESCRIPTION__</p>
    <p id="status" class="note" aria-live="polite">__COUNT__ snapshots from the latest __LIMIT__ first-parent master commits. Commits before Groma setup have no map.</p>
    <details id="about">
      <summary>Read the full architecture description</summary>
      <div id="overview">__OVERVIEW__</div>
    </details>
  </header>
  <iframe id="map" title="Groma architecture map" src="revisions/__HEAD__/"></iframe>
  <script id="history-data" type="application/json">__DATA__</script>
  <script>
    const revisions = JSON.parse(document.getElementById('history-data').textContent);
    const selector = document.getElementById('revision');
    const frame = document.getElementById('map');
    const status = document.getElementById('status');
    const summary = status.textContent;
    function showRevision(updateHistory = false) {
      const params = new URLSearchParams(location.search);
      const requested = updateHistory ? selector.value : params.get('revision');
      const item = revisions.find(revision => revision.id === requested) || revisions[0];
      selector.value = item.id;
      document.getElementById('project-title').textContent = item.title;
      document.getElementById('description').textContent = item.description;
      document.getElementById('overview').textContent = item.overview;
      document.getElementById('commit-link').href = 'https://github.com/Raithlin/gha/commit/' + item.id;
      frame.src = 'revisions/' + item.id + '/';
      status.textContent = (requested && requested !== item.id ? 'That revision is outside the published history; showing latest master. ' : '')
        + (item.id === revisions[0].id ? 'Latest master · ' : 'Historical master commit · ')
        + item.date + ' · ' + summary;
      if (updateHistory) {
        const url = new URL(location.href);
        url.search = item.id === revisions[0].id ? '' : '?revision=' + item.id;
        history.pushState(null, '', url);
      }
    }
    selector.addEventListener('change', () => showRevision(true));
    window.addEventListener('popstate', () => showRevision());
    showRevision();
  </script>
</body>
</html>
'''
    replacements = {'TITLE': html.escape(latest['title']), 'HEAD': latest['id'],
                    'DESCRIPTION': html.escape(latest['description']),
                    'OVERVIEW': html.escape(latest['overview']), 'OPTIONS': options,
                    'DATA': data, 'COUNT': str(len(revisions)), 'LIMIT': str(limit)}
    # One substitution pass prevents hostile metadata from introducing tokens.
    import re
    return re.sub(r'__(\w+)__', lambda match: replacements[match[1]], page)


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('directory', type=Path)
    parser.add_argument('--url', required=True)
    parser.add_argument('--revision', default='HEAD')
    parser.add_argument('--limit', type=int, default=30)
    args = parser.parse_args()
    try:
        revisions = build(Path.cwd(), args.directory.resolve(), args.url,
                          revision=args.revision, limit=args.limit)
    except (ValueError, OSError, subprocess.CalledProcessError) as error:
        parser.exit(1, f'Architecture publication failed: {error}\n')
    print(f'Published {len(revisions)} master snapshots to {args.directory}')


if __name__ == '__main__':
    main()
