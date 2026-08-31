# MOI prototype synchronization

The deployed prototype in `product/moi-platform-prototype/` is a Git subtree of
the `html/` directory from
[`matrixorigin/moi-prototype`](https://github.com/matrixorigin/moi-prototype).
Casebook-only deployment changes are committed on top of that subtree.

## Check for updates

From the repository root, run:

```bash
./scripts/sync-moi-prototype.sh --check
```

Exit status `0` means the prototype is current, `1` means an update is
available, and `2` or greater means the check failed. The first run creates a
blob-filtered upstream cache inside this repository's `.git/` directory;
subsequent runs reuse it.

To check against a local clone instead of GitHub:

```bash
MOI_PROTOTYPE_REPO=../moi-prototype ./scripts/sync-moi-prototype.sh --check
```

## Apply an update

Start from an up-to-date branch with a clean working tree:

```bash
git switch main
git pull
git switch -c fix/sync-moi-prototype-YYYYMMDD
./scripts/sync-moi-prototype.sh
```

The script fetches the upstream branch, splits out only `html/`, and performs a
squashed subtree merge. It never imports upstream `customers/`, `docs/`,
`output/`, or other repository content.

If the merge succeeds, review the generated commit, verify the prototype, and
push the branch.

If Git reports conflicts, preserve casebook-only behavior while incorporating
the upstream feature. Then finish the merge with:

```bash
git add <resolved-files>
git commit
```

To abandon a conflicted synchronization, run `git merge --abort`.

## Change ownership

- Product changes that should benefit every consumer belong in
  `moi-prototype` first and should reach this repository through synchronization.
- Casebook-specific deployment changes belong in this repository.
- Do not copy or `rsync` files between the repositories; doing so bypasses the
  subtree history needed for reliable future merges.
