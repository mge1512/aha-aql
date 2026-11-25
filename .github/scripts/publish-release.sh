#!/bin/bash
# publish-release.sh TAG DIR
#
# Publishes the files of DIR as the GitHub release of TAG.
#
#   1. The tag must name the commit that this run built (GITHUB_SHA).
#   2. If the tag has no release yet, a draft is created. A draft that is
#      there already is used, with the title and the notes that it has.
#   3. Every file of DIR is uploaded, unless the release has it already
#      with the same SHA-256.
#   4. The draft is published.
#
# The script can run again at any time: it goes on where an earlier run
# stopped. It tries every request to GitHub several times, because the
# release API sometimes answers a correct request with a server error.
#
# The script never changes a release that is published. It stops when the
# tag has more than one release or draft, and when a draft has a file that
# DIR does not have.
#
# Environment:
#   GH_TOKEN            token with the permission "contents: write"
#   GITHUB_REPOSITORY   owner/name
#   GITHUB_SHA          the commit that was built
#   RELEASE_TITLE       title of a new draft (default: the tag)
#   RELEASE_NOTES_FILE  file with the notes of a new draft (default: none)
#   RELEASE_TRIES       tries for each request (default: 5)
#   RELEASE_WAIT        seconds before the second try; the time doubles
#                       with each further try (default: 10)
#   RELEASE_DRY_RUN     1: send no request that changes something, and say
#                       what would be done

set -euo pipefail

usage='usage: publish-release.sh TAG DIR'
tag=${1:?$usage}
dir=${2:?$usage}
repo=${GITHUB_REPOSITORY:?GITHUB_REPOSITORY is not set}
built=${GITHUB_SHA:?GITHUB_SHA is not set}
title=${RELEASE_TITLE:-$tag}
notes=${RELEASE_NOTES_FILE:-}
tries=${RELEASE_TRIES:-5}
pause=${RELEASE_WAIT:-10}
dry=${RELEASE_DRY_RUN:-0}

say() { printf '%s\n' "$*" >&2; }
die() { say "error: $*"; exit 1; }

# The tag and the file names become part of request paths and of jq
# filters, so they have plain characters only.
plain() {
    case $1 in
        '' | .* | -* | *[!A-Za-z0-9._-]*) return 1 ;;
    esac
}

# nap N: waits before try N + 1.
nap() { sleep $((pause * (1 << ($1 - 1)))); }

# get PATH FILTER: prints the answer to a GET request, filtered with jq.
get() {
    local n=1 out
    until out=$(gh api "$1" --paginate --jq "$2" < /dev/null); do
        if [ "$n" -ge "$tries" ]; then
            say "GET $1 failed $tries times"
            return 1
        fi
        say "GET $1 failed (try $n of $tries)"
        nap "$n"
        n=$((n + 1))
    done
    [ -z "$out" ] || printf '%s\n' "$out"
}

# check_tag: the tag names the commit that was built.
check_tag() {
    local at
    at=$(get "repos/$repo/commits/refs/tags/$tag" .sha) || die "cannot read the tag $tag"
    [ "$at" = "$built" ] || die "the tag $tag names the commit $at, but this run built $built"
}

# releases: one line "ID DRAFT" for each release or draft of the tag.
releases() {
    get "repos/$repo/releases?per_page=100" \
        ".[] | select(.tag_name == \"$tag\") | \"\(.id) \(.draft)\""
}

# assets ID: one line "NAME DIGEST" for each file of the release. DIGEST is
# "-" for a file whose upload is not complete.
assets() {
    get "repos/$repo/releases/$1/assets?per_page=100" \
        '.[] | "\(.name) \(if .state == "uploaded" then (.digest // "-") else "-" end)"' |
        LC_ALL=C sort
}

# wanted: one line "NAME DIGEST" for each file of DIR.
wanted() {
    local f name
    for f in "$dir"/*; do
        name=${f##*/}
        [ -f "$f" ] || die "no files in $dir, or $name is not a file"
        plain "$name" || die "this script does not accept the file name '$name'"
        printf '%s sha256:%s\n' "$name" "$(sha256sum < "$f" | cut -d' ' -f1)"
    done | LC_ALL=C sort
}

# the_release: prints "ID DRAFT" for the release of the tag. Creates a
# draft when the tag has none. After a request that failed it looks again
# first, because GitHub can create the draft and still report an error.
the_release() {
    local n=1 found count id
    while :; do
        found=$(releases) || die "cannot list the releases"
        count=$(printf '%s' "$found" | grep -c . || true)
        if [ "$count" -eq 1 ]; then
            printf '%s\n' "$found"
            return 0
        fi
        [ "$count" -eq 0 ] || die "the tag $tag has $count releases or drafts. Delete all but one, then run this job again: https://github.com/$repo/releases"
        if [ "$dry" = 1 ]; then
            say "would create a draft release for $tag"
            printf 'new true\n'
            return 0
        fi
        [ "$n" -le "$tries" ] || die "cannot create a draft release for $tag"
        [ "$n" -eq 1 ] || nap $((n - 1))
        say "creating a draft release for $tag (try $n of $tries)"
        if id=$(gh api --method POST "repos/$repo/releases" -f tag_name="$tag" \
            -f name="$title" -f body="$body" -F draft=true --jq .id < /dev/null); then
            case $id in
                '' | *[!0-9]*) ;;
                *)
                    printf '%s true\n' "$id"
                    return 0
                    ;;
            esac
        fi
        say "the request failed"
        n=$((n + 1))
    done
}

# upload_files ID: uploads every file that the release does not have with
# the same content. A file of the same name that is incomplete or has
# other content is replaced. Leaves the last list of files in $tmp/have.
upload_files() {
    local id=$1 n=1 name
    while :; do
        assets "$id" > "$tmp/have" || die "cannot list the files of the release"
        LC_ALL=C comm -23 "$tmp/want" "$tmp/have" | cut -d' ' -f1 > "$tmp/todo"
        [ -s "$tmp/todo" ] || return 0
        if [ "$dry" = 1 ]; then
            say "would upload: $(paste -sd' ' "$tmp/todo")"
            return 0
        fi
        [ "$n" -le "$tries" ] || die "after $tries tries the release does not have: $(paste -sd' ' "$tmp/todo")"
        [ "$n" -eq 1 ] || nap $((n - 1))
        while read -r name <&3; do
            say "uploading $name (try $n of $tries)"
            gh release upload "$tag" "$dir/$name" --clobber --repo "$repo" < /dev/null ||
                say "the request failed"
        done 3< "$tmp/todo"
        n=$((n + 1))
    done
}

# publish ID: publishes the draft. It looks first, because GitHub can
# publish the draft and still report an error.
publish() {
    local id=$1 n=1 found
    while :; do
        found=$(releases) || die "cannot list the releases"
        case $found in
            "$id false") return 0 ;;
            "$id true") ;;
            *) die "the releases of $tag changed while this job ran: $(printf '%s' "$found" | paste -sd, -)" ;;
        esac
        if [ "$dry" = 1 ]; then
            say "would publish the draft"
            return 0
        fi
        [ "$n" -le "$tries" ] || die "cannot publish the release of $tag. The draft has all files; run this job again"
        [ "$n" -eq 1 ] || nap $((n - 1))
        say "publishing the release of $tag (try $n of $tries)"
        gh api --method PATCH "repos/$repo/releases/$id" -F draft=false --silent < /dev/null ||
            say "the request failed"
        n=$((n + 1))
    done
}

plain "$tag" || die "this script does not accept the tag '$tag'"
[ -d "$dir" ] || die "no directory $dir"
body=
if [ -n "$notes" ]; then
    [ -f "$notes" ] || die "no file $notes"
    body=$(cat "$notes")
fi

tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT

wanted > "$tmp/want" || exit 1
[ -s "$tmp/want" ] || die "no files in $dir"

check_tag
state=$(the_release)
id=${state% *}
draft=${state#* }

if [ "$draft" = false ]; then
    assets "$id" > "$tmp/have" || die "cannot list the files of the release"
    if cmp -s "$tmp/want" "$tmp/have"; then
        say "the release of $tag is published already, with exactly these files. Nothing to do"
        exit 0
    fi
    die "the release of $tag is published already, and its files are not the files of this run. A published release is not changed"
fi

if [ "$id" = new ]; then
    say "would upload: $(cut -d' ' -f1 "$tmp/want" | paste -sd' ' -)"
    say "would publish the draft"
    exit 0
fi

upload_files "$id"
extra=$(cut -d' ' -f1 "$tmp/have" | LC_ALL=C sort |
    LC_ALL=C comm -13 <(cut -d' ' -f1 "$tmp/want" | LC_ALL=C sort) - | paste -sd' ' -)
[ -z "$extra" ] || die "the draft has files that this run did not build: $extra. Delete them, then run this job again: https://github.com/$repo/releases"

check_tag
publish "$id"
[ "$dry" = 1 ] || say "published: https://github.com/$repo/releases/tag/$tag"
