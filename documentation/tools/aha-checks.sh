#!/bin/bash
# aha-checks.sh - the checks 1 to 26 of the aha-aql notes (section 6)
#
# Sends the requests that mcp-server-aha-aql 0.20.0 would send, with curl,
# and prints what Aha! answers. Use a product that can be thrown away.
#
# Usage:
#   aha-checks.sh products [TEXT]
#                         list the products that the API key can see, with
#                         key and name. With TEXT: only the products that
#                         have TEXT in the key or in the name
#   aha-checks.sh show    print the requests of both parts and send nothing
#   aha-checks.sh read    the checks that only read: 1, 8, 10, 11, 12, 17 and
#                         19 to 22. Most requests go to the test product.
#                         Page 1 of the account-wide lists is read too,
#                         and one epic of the account with its product
#   aha-checks.sh write   the checks that create and change records in the
#                         test product: 2 to 7, 13 to 16, 18 and 21 to 26.
#                         The script deletes these records at the end
#
# Settings, from the environment:
#   AHA_SUBDOMAIN       the account name, as in https://NAME.aha.io
#   AHA_API_KEY_FILE    a file with the API key (default ~/.config/aha/api-key),
#                       or AHA_API_KEY with the key itself
#   AHA_CHECK_PRODUCT   the test product: its key, its numeric id or its name,
#                       as the server accepts them. The key is the part before
#                       the first hyphen in the ids of its records: DEMO in
#                       DEMO-123. A name must be the name of one product only
#   AHA_CHECK_SHORT=1   write part: only the checks 3, 25 and 26. The script
#                       then creates 2 releases, 1 initiative and 1 feature
#   AHA_CHECK_KEEP=1    write part: keep the records at the end
#   AHA_CHECK_PAUSE=1   write part: wait for Enter before the records are
#                       deleted, so that you can look at them in Aha!
#   AHA_CHECK_EDITABLE=1
#                       write part: also write to a tags field that takes new
#                       values. Aha! can keep such values as choices of the
#                       field in every product
#   AHA_CHECK_FIELD_NOT_IN_LAYOUT
#                       check 18: the key of a custom field of features that
#                       the layout of the test product does not have
#
# Needs bash 4, curl and jq.
#
# The output has no API key. It has names, ids and e-mail addresses of the
# account. Checks 9 and the second half of check 8 need a person; the script
# says what to do.
#
# What the write part can leave behind: the tags aql-check-1 and aql-check-2,
# which it sets on its feature, can stay in the list of tags of the account.
#
# License: MIT

set -u

MODE=${1:-}
FILTER=${2:-}
case "$MODE" in
    show|read|write|products) ;;
    *) sed -n '2,/^# Needs/p' "$0" | sed 's/^# \{0,1\}//'; exit 2 ;;
esac

for tool in curl jq; do
    command -v "$tool" > /dev/null 2>&1 || { echo "error: $tool is not installed" >&2; exit 2; }
done

SUB=${AHA_SUBDOMAIN:-}
PRODUCT=${AHA_CHECK_PRODUCT:-}
if [ "$MODE" = show ]; then
    SUB=${SUB:-ACCOUNT}
    PRODUCT=${PRODUCT:-PRODUCT}
fi
[ -n "$SUB" ]     || { echo "error: set AHA_SUBDOMAIN" >&2; exit 2; }
if [ -z "$PRODUCT" ] && [ "$MODE" != products ]; then
    echo "error: set AHA_CHECK_PRODUCT to the key or the name of the test product." >&2
    echo "       '$0 products' lists the products of the account" >&2
    exit 2
fi
if [ "$MODE" != show ]; then
    case "$SUB" in *[!a-z0-9-]*) echo "error: AHA_SUBDOMAIN is the account name only, in lower case" >&2; exit 2 ;; esac
fi

BASE="https://$SUB.aha.io/api/v1"
# For a self-test of this script against a local stand-in. Only a loopback
# address is accepted, so that the key cannot go to another host.
case "${AHA_CHECK_BASE:-}" in
    '') ;;
    http://127.0.0.1:*|http://localhost:*) BASE=$AHA_CHECK_BASE ;;
    *) echo "error: AHA_CHECK_BASE must be a loopback address" >&2; exit 2 ;;
esac

TMP=$(mktemp -d) || exit 1
trap 'rm -rf "$TMP"' EXIT
OUT=$TMP/answer.json
CFG=$TMP/curl.cfg
BADCFG=$TMP/curl-bad.cfg
PAUSE=0.2
STATUS=000
PARAMS=()
STAMP=$(date +%Y%m%d-%H%M%S)
PREFIX="aql-check $STAMP"
TAGS="aql-check-1,aql-check-2"      # in the order in which they sort

# The key goes into a curl configuration file that only this user can read.
# It is never an argument of a command.
if [ "$MODE" != show ]; then
    KEY=${AHA_API_KEY:-}
    if [ -z "$KEY" ]; then
        KEYFILE=${AHA_API_KEY_FILE:-$HOME/.config/aha/api-key}
        [ -r "$KEYFILE" ] || { echo "error: no API key: set AHA_API_KEY_FILE or AHA_API_KEY" >&2; exit 2; }
        KEY=$(tr -d '\r\n' < "$KEYFILE")
    fi
    [ -n "$KEY" ] || { echo "error: the API key is empty" >&2; exit 2; }
    ( umask 077
      esc=$(printf '%s' "$KEY" | sed 's/\\/\\\\/g; s/"/\\"/g')
      printf 'header = "Authorization: Bearer %s"\n' "$esc" > "$CFG"
      printf 'header = "Authorization: Bearer wrong-key-of-check-8"\n' > "$BADCFG"
      for f in "$CFG" "$BADCFG"; do
          printf 'header = "Accept: application/json"\n' >> "$f"
          printf 'user-agent = "aha-aql-checks/0.20.0 (https://github.com/mge1512/aha-aql)"\n' >> "$f"
      done )
    unset KEY
fi

# line VERDICT NO TEXT. The verdicts:
#   PASS  as the specification expects
#   FAIL  the request failed, or the answer has a form that the server
#         would refuse
#   NOTE  the request works, and the answer differs from what the
#         specification expects
#   INFO  a fact for the reader, without a verdict
#   SKIP  the account has nothing to run the check on
declare -A COUNT=([PASS]=0 [FAIL]=0 [NOTE]=0 [INFO]=0 [SKIP]=0)
line() { COUNT[$1]=$(( ${COUNT[$1]:-0} + 1 )); printf '%-5s %-5s %s\n' "$1" "$2" "$3"; }
detail() { printf '            %s\n' "$*"; }
details() { local l; while IFS= read -r l; do [ -z "$l" ] || detail "$l"; done; }
j() { jq -r "$@" "$OUT" 2> /dev/null; }

# req METHOD ROUTE [JSON-BODY]
# Query parameters come from the array PARAMS and are percent-encoded, as
# the server does it. Sets STATUS and writes the body of the answer to $OUT.
req() {
    local method=$1 route=$2 body=${3-} cfg=${REQCFG:-$CFG} args=() p
    if [ "$MODE" = show ]; then
        printf '      %s %s' "$method" "$route"
        if [ ${#PARAMS[@]} -gt 0 ]; then printf '?%s' "$(IFS='&'; echo "${PARAMS[*]}")"; fi
        printf '\n'
        if [ -n "$body" ]; then printf '          body: %s\n' "$body"; fi
        STATUS=000; : > "$OUT"; PARAMS=(); return 0
    fi
    if [ ${#PARAMS[@]} -gt 0 ]; then
        args+=(-G)
        for p in "${PARAMS[@]}"; do args+=(--data-urlencode "$p"); done
    fi
    if [ -n "$body" ]; then
        args+=(-H 'Content-Type: application/json' --data-binary "$body")
    fi
    STATUS=$(curl -sS --config "$cfg" --max-time 60 -o "$OUT" -w '%{http_code}' \
                  -X "$method" "${args[@]}" "$BASE$route") || STATUS=000
    PARAMS=()
    sleep "$PAUSE"
}

ok() { case "$STATUS" in 2??) return 0 ;; *) return 1 ;; esac; }

# The first 200 characters of an answer, for a status that was not expected.
excerpt() { head -c 200 "$OUT" | tr '\n' ' '; }

# ---------------------------------------------------------------------------
# The products of the account, and the test product
# ---------------------------------------------------------------------------

PRODUCTS=$TMP/products.jsonl    # one line per product: id, key, name, line
NPRODUCTS=0
MOREPAGES=
PID="{product id}"
PNAME=
NOID=

# The comparison form of the specification: no surrounding white space and
# no difference of case.
JQ_CF='def trim: if startswith(" ") then .[1:] | trim elif endswith(" ") then .[:-1] | trim else . end;
       def cf: tostring | trim | ascii_downcase;'

# All pages of the product list, as resolve-product of the specification
# reads it.
load_products() {
    local page=1 pages=1
    : > "$PRODUCTS"
    while [ "$page" -le "$pages" ]; do
        if [ "$page" -gt 15 ]; then MOREPAGES=$pages; break; fi
        PARAMS=("fields=reference_prefix,name,product_line" "per_page=200" "page=$page"); req GET /products
        ok || { echo "error: GET /products: status $STATUS: $(excerpt)" >&2; exit 1; }
        if [ "$(j '.products | type')" != array ]; then
            echo "error: GET /products: no list in the member products. Members: $(j 'keys | join(", ")')" >&2
            exit 1
        fi
        jq -c '.products[] | {id: (.id // "" | tostring), key: (.reference_prefix // "" | tostring),
                              name: (.name // "" | tostring), line: (.product_line == true)}' "$OUT" >> "$PRODUCTS"
        pages=$(j '.pagination.total_pages // 1')
        case "$pages" in ''|*[!0-9]*) pages=1 ;; esac
        page=$((page + 1))
    done
    NPRODUCTS=$(grep -c . "$PRODUCTS")
    [ -z "$MOREPAGES" ] || echo "note: the product list has $MOREPAGES pages. Only the first 15 were read" >&2
}

# match_products HOW VALUE: prints the products that the value means.
#   id    the key or the numeric id is the value
#   name  the name is the value
#   part  the value is inside the key or inside the name; no value: all
match_products() {
    jq -c --arg how "$1" --arg v "$2" "$JQ_CF"'
        ($v | cf) as $x
        | select(if   $how == "id"   then (.key | cf) == $x or .id == $x
                 elif $how == "name" then (.name | cf) == $x
                 else $x == "" or ((.key | cf) | contains($x)) or ((.name | cf) | contains($x)) end)' "$PRODUCTS"
}

# print_products: prints the products on standard input, one line each
print_products() {
    jq -r '"\(.key)\t\(.name)\(if .line then "  [product line]" else "" end)"' | LC_ALL=C sort -f |
        while IFS=$'\t' read -r k n; do printf '  %-12s %s\n' "$k" "$n"; done
}

# find_product: finds the test product as resolve-product finds a product:
# by its key or its numeric id first, and then by its name. Other than the
# server, the script stops when several products have that name.
find_product() {
    local found near n how="its key"
    if [ "$MODE" = show ]; then
        PARAMS=("fields=reference_prefix,name,product_line" "per_page=200" "page=1"); req GET /products; return 0
    fi
    load_products
    found=$(match_products id "$PRODUCT")
    if [ -z "$found" ]; then found=$(match_products name "$PRODUCT"); how="its name"; fi
    n=$(printf '%s\n' "$found" | grep -c .)
    if [ "$n" -eq 0 ]; then
        echo "error: the account $SUB has no product with the key, the id or the name '$PRODUCT'" >&2
        echo "       ($NPRODUCTS products are visible with this API key)" >&2
        near=$(match_products part "$PRODUCT")
        if [ -n "$near" ]; then
            echo "Products with that text in the key or in the name:" >&2
            printf '%s\n' "$near" | print_products >&2
        else
            echo "'$0 products TEXT' lists the products with TEXT in the key or in the name." >&2
        fi
        exit 1
    fi
    if [ "$n" -gt 1 ]; then
        echo "error: $n products of the account $SUB have the name '$PRODUCT'. Set AHA_CHECK_PRODUCT to a key:" >&2
        printf '%s\n' "$found" | print_products >&2
        exit 1
    fi
    PID=$(printf '%s\n' "$found" | jq -r '.id')
    PNAME=$(printf '%s\n' "$found" | jq -r '.name')
    if [ "$(printf '%s\n' "$found" | jq -r '.line')" = true ]; then
        echo "error: '$PRODUCT' is a product line, a group of products. The checks need a product" >&2
        echo "       that has releases and features. Take one of its products." >&2
        exit 1
    fi
    [ "$how" = "its key" ] || echo "'$PRODUCT' is the name of the product with the key $(printf '%s\n' "$found" | jq -r '.key')"
    PRODUCT=$(printf '%s\n' "$found" | jq -r '.key')
    # The specification expects the member id in every record of a list.
    # Without it, the routes get the key.
    if [ -z "$PID" ]; then NOID=1; PID=$PRODUCT; fi
}

# first_ref ROUTE MEMBER PLACEHOLDER -> sets FIRST to the reference number of
# the first record, or to nothing
first_ref() {
    FIRST=
    PARAMS=("fields=reference_num,name" "per_page=200" "page=1"); req GET "$1"
    if [ "$MODE" = show ]; then FIRST="{$3}"; return 0; fi
    ok && FIRST=$(jq -r --arg m "$2" '.[$m][0].reference_num // empty' "$OUT")
    return 0
}

# ---------------------------------------------------------------------------
# Check 1: every list route accepts fields and per_page=200, and the records
# have the members that the catalog names
# ---------------------------------------------------------------------------

# The API sources of the catalog of version 0.20.0, without the heavy fields.
S_PRODUCTS="reference_prefix name product_line parent_id created_at url"
S_RELEASES="reference_num name release_date start_date workflow_status.name parking_lot released product_id progress created_at url"
S_FEATURES="reference_num name workflow_status.name release_reference_num initiative_reference_num epic_reference_num assigned_to_user.email product_id start_date due_date tags score progress created_at updated_at url"
S_EPICS="reference_num name workflow_status.name release.reference_num initiative_reference_num assigned_to_user.email product_id start_date due_date progress created_at updated_at url"
S_REQUIREMENTS="reference_num name workflow_status.name feature.reference_num assigned_to_user.email original_estimate work_done created_at url"
S_INITIATIVES="reference_num name workflow_status.name product_id start_date end_date progress value effort color presented created_at updated_at url"
S_GOALS="reference_num name success_metric.name success_metric.workflow_status.name product_id progress value effort created_at updated_at url"
S_IDEAS="reference_num name workflow_status.name votes product_id feature.reference_num visibility created_at updated_at url"
S_COMMENTS="id body user.name commentable.url created_at url"
S_USERS="id name email created_at"

# fields_of SOURCES: the parameter fields for these sources. Of a source with
# a dot it has the part before the dot, as plan-fetch says.
fields_of() {
    local s root out=,
    for s in $1; do
        root=${s%%.*}
        case "$out" in *",$root,"*) ;; *) out="$out$root," ;; esac
    done
    out=${out#,}
    printf '%s' "${out%,}"
}

# What the records of one page say about the sources. A line that starts
# with N is a difference from the specification, a line with I is a fact.
# shellcheck disable=SC2016  # a jq program, not shell
JQ_SOURCES='
def haspath($p): reduce $p[] as $k ({o: ., ok: true};
    if .ok and (.o | type) == "object" and (.o | has($k)) then {o: .o[$k], ok: true}
    else {o: null, ok: false} end) | .ok;
.[$m] as $recs | ($recs | length) as $n
| [ $src | split(" ")[] | select(length > 0) | . as $s | ($s | split(".")) as $p
    | { s: $s, root: $p[0], dotted: (($p | length) > 1),
        top:  ([$recs[] | select(has($p[0]))] | length),
        obj:  ([$recs[] | select(has($p[0]) and (.[$p[0]] | type) == "object")] | length),
        full: ([$recs[] | select(haspath($p))] | length) } ] as $t
| ([$t[] | select(.top == 0) | .root] | unique) as $none
| ([$t[] | select(.dotted and .obj > 0 and .full == 0) | .s]) as $bad
| ([$t[] | select(.top > 0 and .top < $n) | "\(.root) (\(.top) of \($n))"] | unique) as $some
| ([$t[] | select(.dotted and .top > 0 and .obj == 0) | .s]) as $open
| (if ($none | length) > 0 then "N no record has the member: " + ($none | join(", ")) else empty end),
  (if ($bad  | length) > 0 then "N the first member is an object, and no record has the source: " + ($bad | join(", ")) else empty end),
  (if ($some | length) > 0 then "I only some records have the member: " + ($some | join(", ")) else empty end),
  (if ($open | length) > 0 then "I not shown by this page, because the first member is an object in no record: " + ($open | join(", ")) else empty end)'

# list_check ROUTE MEMBER SOURCES
list_check() {
    local route=$1 member=$2 src=$3 n pag noid notes verdict=PASS
    case "$route" in */NONE/*) line SKIP 1 "GET $route: no such parent record was found"; return ;; esac
    PARAMS=("fields=$(fields_of "$src")" "per_page=200" "page=1"); req GET "$route"
    [ "$MODE" = show ] && return
    if ! ok; then line FAIL 1 "GET $route: status $STATUS: $(excerpt)"; return; fi
    if ! jq -e --arg m "$member" '(.[$m] | type) == "array"' "$OUT" > /dev/null 2>&1; then
        line FAIL 1 "GET $route: no list in the member $member. Members: $(j 'keys | join(", ")')"; return
    fi
    n=$(jq -r --arg m "$member" '.[$m] | length' "$OUT")
    pag=$(j 'if (.pagination | type) == "object" then "total_records \(.pagination.total_records), total_pages \(.pagination.total_pages)" else "no pagination" end')
    [ "$pag" = "no pagination" ] && verdict=FAIL
    noid=$(jq -r --arg m "$member" '[.[$m][] | select(has("id") | not)] | length' "$OUT" 2> /dev/null)
    if [ "${noid:-0}" -gt 0 ]; then verdict=FAIL; pag="$pag. $noid records have no member id"; fi
    notes=
    [ "$n" -gt 0 ] && notes=$(jq -r --arg m "$member" --arg src "$src" "$JQ_SOURCES" "$OUT" 2> /dev/null)
    case "$notes" in N\ *|*$'\n'N\ *) [ "$verdict" = PASS ] && verdict=NOTE ;; esac
    line "$verdict" 1 "GET $route: $n records, $pag"
    [ -z "$notes" ] || printf '%s\n' "$notes" | sed 's/^[NI] //' | details
}

check_1() {
    local rel feat epic init
    first_ref "/products/$PID/releases" releases release;          rel=${FIRST:-NONE}
    first_ref "/products/$PID/features" features feature;          feat=${FIRST:-NONE}
    first_ref "/products/$PID/epics" epics epic;                   epic=${FIRST:-NONE}
    first_ref "/products/$PID/initiatives" initiatives initiative; init=${FIRST:-NONE}
    first_ref "/products/$PID/goals" goals goal;                   READ_GOAL=${FIRST:-}
    list_check /products products "$S_PRODUCTS"
    list_check "/products/$PID/releases" releases "$S_RELEASES"
    list_check "/products/$PID/features" features "$S_FEATURES"
    list_check "/releases/$rel/features" features "$S_FEATURES"
    list_check "/epics/$epic/features" features "$S_FEATURES"
    list_check "/initiatives/$init/features" features "$S_FEATURES"
    list_check /features features "$S_FEATURES"
    list_check "/products/$PID/epics" epics "$S_EPICS"
    list_check "/releases/$rel/epics" epics "$S_EPICS"
    list_check "/initiatives/$init/epics" epics "$S_EPICS"
    list_check /epics epics "$S_EPICS"
    list_check "/features/$feat/requirements" requirements "$S_REQUIREMENTS"
    list_check "/products/$PID/initiatives" initiatives "$S_INITIATIVES"
    list_check /initiatives initiatives "$S_INITIATIVES"
    list_check "/products/$PID/goals" goals "$S_GOALS"
    list_check /goals goals "$S_GOALS"
    list_check "/products/$PID/ideas" ideas "$S_IDEAS"
    list_check /ideas ideas "$S_IDEAS"
    list_check "/features/$feat/comments" comments "$S_COMMENTS"
    list_check "/products/$PID/comments" comments "$S_COMMENTS"
    list_check /users users "$S_USERS"
    READ_FEATURE=; [ "$feat" = NONE ] || READ_FEATURE=$feat
}

# ---------------------------------------------------------------------------
# The JSON forms of single answers (INTERFACES)
# ---------------------------------------------------------------------------

check_forms() {
    local t
    if [ -n "${READ_FEATURE:-}" ]; then
        req GET "/features/$READ_FEATURE"
        if [ "$MODE" != show ]; then
            if ok && [ "$(j '.feature | type')" = object ]; then
                line PASS 19 "GET /features/$READ_FEATURE: the record is in the member feature"
                detail "description is $(j '.feature.description | if type == "object" then "an object with the members " + (keys | join(", ")) else type end')"
                detail "tags is $(j '.feature.tags | type'); custom_fields is $(j '.feature.custom_fields | type')"
                detail "release_reference_num is $(j '.feature.release_reference_num | type'); workflow_status.name is $(j '.feature.workflow_status.name | type')"
            else
                line FAIL 19 "GET /features/$READ_FEATURE: status $STATUS. Members: $(j 'keys | join(", ")')"
            fi
        fi
    else
        line SKIP 19 "one feature: the test product has no feature yet. Run the write part with AHA_CHECK_KEEP=1 first"
    fi
    if [ -n "${READ_GOAL:-}" ]; then
        req GET "/goals/$READ_GOAL"
        if [ "$MODE" != show ]; then
            if ok && [ "$(j '.goal | type')" = object ]; then
                line PASS 19 "GET /goals/$READ_GOAL: the record is in the member goal"
                detail "success_metric is $(j '.goal.success_metric | if type == "object" then "an object with the members " + (keys | join(", ")) else type end')"
                detail "success_metric.workflow_status.name is $(j '.goal.success_metric.workflow_status.name | type'); the member workflow_status is $(j 'if (.goal | has("workflow_status")) then "there" else "absent" end')"
            else
                line FAIL 19 "GET /goals/$READ_GOAL: status $STATUS. Members: $(j 'keys | join(", ")')"
            fi
        fi
    fi
    req GET "/products/$PID"
    if [ "$MODE" != show ]; then
        cp "$OUT" "$TMP/product.json"
        if ok && [ "$(j '.product | type')" = object ]; then
            t=$(j '.product.custom_fields | type')
            if [ "$t" = array ]; then line PASS 20 "GET /products/$PID: member product, with custom_fields ($(j '.product.custom_fields | length') entries)"
            else line FAIL 20 "GET /products/$PID: member product, and custom_fields is $t. SELECT * on a product needs that member"; fi
        else
            line FAIL 20 "GET /products/$PID: status $STATUS. Members: $(j 'keys | join(", ")')"
        fi
    fi
    req GET "/products/$PID/workflows"
    if [ "$MODE" != show ]; then
        cp "$OUT" "$TMP/workflows.json"
        if ok && [ "$(j '.workflows | type')" = array ]; then
            line PASS 19 "GET /products/$PID/workflows: $(j '.workflows | length') workflows"
            detail "record types: $(j '[.workflows[].statusable_type] | join(", ")')"
            detail "a status has: $(j '.workflows[0].workflow_statuses[0] | keys | join(", ")')"
        else
            line FAIL 19 "GET /products/$PID/workflows: status $STATUS. Members: $(j 'keys | join(", ")')"
        fi
    fi
    req GET /me
    if [ "$MODE" != show ]; then
        if ok && [ "$(j '.user.email | type')" = string ]; then line PASS 19 "GET /me: the record is in the member user"
        else line FAIL 19 "GET /me: status $STATUS. Members: $(j 'keys | join(", ")')"; fi
    fi
}

# ---------------------------------------------------------------------------
# Check 21: the record of a product names the workflow of each record kind
# Check 22: an epic names its release in the object release
# Both come from the runs of 2026-11-24.
# ---------------------------------------------------------------------------

# kinds_report PRODUCT-FILE WORKFLOWS-FILE: one line per workflow that the
# record of the product names
kinds_report() {
    jq -r --slurpfile w "$2" '
        ($w[0].workflows // []) as $list
        | .product | to_entries[]
        | select((.key | endswith("_workflow")) and ((.value | type) == "object"))
        | .key as $k | .value as $v
        | ([$list[] | select((.id | tostring) == ($v.id | tostring))] | length) as $in
        | "\($k): record type \($v.statusable_type // "not given"), \(if $in > 0 then "in" else "not in" end) the list of workflows"' "$1" 2> /dev/null
}

# workflow_type_of WORKFLOWS-FILE STATUS-ID: the record types of the
# workflows that have this status
workflow_type_of() {
    jq -r --arg s "$2" '[.workflows[]? | select(any(.workflow_statuses[]?; (.id | tostring) == $s)) | .statusable_type] | join(", ")' "$1" 2> /dev/null
}

# other_status WORKFLOWS-FILE STATUS-ID: the name of another status of the
# workflow that has this status
other_status() {
    jq -r --arg s "$2" '[.workflows[]? | select(any(.workflow_statuses[]?; (.id | tostring) == $s))][0].workflow_statuses
        | map(select((.id | tostring) != $s))[0].name // empty' "$1" 2> /dev/null
}

check_21() {
    local kinds state bad
    [ "$MODE" = show ] && return
    [ -s "$TMP/product.json" ] && [ -s "$TMP/workflows.json" ] || { line SKIP 21 "the product and its workflows were not read"; return; }
    kinds=$(kinds_report "$TMP/product.json" "$TMP/workflows.json")
    if [ -z "$kinds" ]; then
        line NOTE 21 "the record of the product names no workflow per record kind: no member ends in _workflow. The server then knows no status in this product"
        return
    fi
    # The entities with a status, and the member of the product record
    # that names their workflow (entity statuses of the specification).
    state=$(jq -r --slurpfile w "$TMP/workflows.json" '
        ($w[0].workflows // []) as $list | .product as $p
        | [["releases", "release_workflow"], ["features", "feature_workflow"], ["epics", "epic_workflow"],
           ["requirements", "requirement_workflow"], ["initiatives", "initiative_workflow"],
           ["goals", "strategic_imperative_workflow"], ["ideas", "idea_workflow"]][]
        | . as [$entity, $member]
        | ($p[$member] | if type == "object" then (.id | tostring) else "" end) as $id
        | if $id == "" then "\($entity): the product has no member \($member)"
          elif ([$list[] | select((.id | tostring) == $id)] | length) == 0 then "\($entity): the workflow of \($member) is not in the list of workflows"
          else empty end' "$TMP/product.json" 2> /dev/null)
    bad=$(printf '%s\n' "$state" | grep -c .)
    if [ "$bad" -eq 0 ]; then
        line PASS 21 "the record of the product names a workflow of its list for each of the 7 entities with a status"
    else
        line NOTE 21 "the server finds no status for $bad of the 7 entities with a status:"
        printf '%s\n' "$state" | details
    fi
    line INFO 21 "the members of the product record that name a workflow:"
    printf '%s\n' "$kinds" | details
}

check_22() {
    local n with ref pid sid t same
    PARAMS=("fields=reference_num,release" "per_page=200" "page=1"); req GET /epics
    if [ "$MODE" = show ]; then
        req GET "/epics/{first epic of that list}"
        req GET "/products/{product of that epic}"
        req GET "/products/{product of that epic}/workflows"
        return
    fi
    if ! ok || [ "$(j '.epics | type')" != array ]; then line FAIL 22 "GET /epics: status $STATUS: $(excerpt)"; return; fi
    n=$(j '.epics | length')
    if [ "$n" -eq 0 ]; then line SKIP 22 "this API key sees no epic in the account"; return; fi
    with=$(j '[.epics[] | select((.release | type) == "object" and (.release.reference_num | type) == "string")] | length')
    if [ "$with" -eq "$n" ]; then line PASS 22 "GET /epics with fields=reference_num,release: all $n records have release.reference_num"
    else line NOTE 22 "GET /epics with fields=reference_num,release: only $with of $n records have release.reference_num"; fi
    ref=$(j '.epics[0].reference_num // empty')
    [ -n "$ref" ] || return
    # From here on the output has no id and no name of that epic, because it
    # can be in any product of the account.
    req GET "/epics/$ref"
    if ! ok || [ "$(j '.epic | type')" != object ]; then line FAIL 22 "GET /epics/{id} for the first epic of the list: status $STATUS. Members: $(j 'keys | join(", ")')"; return; fi
    line INFO 22 "an epic that is read by its id: release.reference_num is $(j '.epic.release.reference_num | type'), the member release_reference_num is $(j 'if (.epic | has("release_reference_num")) then "there" else "absent" end'), the member color is $(j 'if (.epic | has("color")) then "there" else "absent" end')"
    pid=$(j '.epic.product_id // empty'); sid=$(j '.epic.workflow_status.id // empty')
    [ -n "$pid" ] && [ -n "$sid" ] || { line NOTE 21 "that epic has no product_id or no workflow_status.id"; return; }
    req GET "/products/$pid"; cp "$OUT" "$TMP/epic-product.json"
    ok || { line FAIL 21 "GET /products/{id} for the product of that epic: status $STATUS"; return; }
    req GET "/products/$pid/workflows"; cp "$OUT" "$TMP/epic-workflows.json"
    ok || { line FAIL 21 "GET /products/{id}/workflows for the product of that epic: status $STATUS"; return; }
    t=$(workflow_type_of "$TMP/epic-workflows.json" "$sid")
    same=$(jq -r --arg s "$sid" --slurpfile w "$TMP/epic-workflows.json" '
        (.product.epic_workflow.id // "" | tostring) as $id
        | if $id == "" then "none"
          elif ([$w[0].workflows[]? | select((.id | tostring) == $id) | select(any(.workflow_statuses[]?; (.id | tostring) == $s))] | length) > 0 then "yes"
          else "no" end' "$TMP/epic-product.json" 2> /dev/null)
    case "$same" in
        yes)  line PASS 21 "an epic of the account: its status is in the workflow that its product names as epic_workflow (record type ${t:-unknown})" ;;
        none) line NOTE 21 "an epic of the account: its product names no epic_workflow" ;;
        *)    line NOTE 21 "an epic of the account: its status is not in the workflow that its product names as epic_workflow. Its workflow has the record type: ${t:-none of the list}" ;;
    esac
}

# ---------------------------------------------------------------------------
# Check 8: a wrong key
# ---------------------------------------------------------------------------

check_8() {
    REQCFG=$BADCFG req GET /me
    [ "$MODE" = show ] && return
    case "$STATUS" in
        401|403) line PASS 8 "a wrong key gets the status $STATUS. The server treats 401 and 403 alike" ;;
        *)       line FAIL 8 "a wrong key gets the status $STATUS: $(excerpt)" ;;
    esac
    line INFO 8 "the status 429 is not provoked. When you see one, note whether it has the header X-Ratelimit-Reset"
}

# ---------------------------------------------------------------------------
# Checks 10, 11, 12, 17: custom fields, reading
# ---------------------------------------------------------------------------

check_10_11() {
    PARAMS=("per_page=200" "page=1"); req GET /custom_field_definitions
    [ "$MODE" = show ] && return
    cp "$OUT" "$TMP/definitions.json"
    case "$STATUS" in
        200)
            if [ "$(j '.custom_field_definitions | type')" != array ]; then
                line FAIL 10 "GET /custom_field_definitions: no list in the member custom_field_definitions. Members: $(j 'keys | join(", ")')"; return
            fi
            line PASS 10 "GET /custom_field_definitions: $(j '.custom_field_definitions | length') definitions, $(j 'if (.pagination | type) == "object" then "with pagination: total_pages \(.pagination.total_pages)" else "one answer without pagination" end')"
            detail "members of a definition: $(j '.custom_field_definitions[0] | keys | join(", ")')"
            line INFO 11 "record type, type and number of definitions:"
            j '.custom_field_definitions[] | [.custom_fieldable_type, .type] | @tsv' | sort | uniq -c | sed 's/^/            /'
            ;;
        403|404)
            line INFO 10 "GET /custom_field_definitions: status $STATUS. This key cannot read the definitions, and the server then works without custom fields"
            ;;
        *)  line FAIL 10 "GET /custom_field_definitions: status $STATUS: $(excerpt)" ;;
    esac
}

# cf_list_check NO ROUTE MEMBER IDFIELD
cf_list_check() {
    local no=$1 route=$2 member=$3 idf=$4 n with
    PARAMS=("fields=$idf,custom_fields" "per_page=200" "page=1"); req GET "$route"
    [ "$MODE" = show ] && return
    if ! ok; then line FAIL "$no" "GET $route with custom_fields: status $STATUS: $(excerpt)"; return; fi
    n=$(jq -r --arg m "$member" '.[$m] | length' "$OUT" 2> /dev/null)
    with=$(jq -r --arg m "$member" '[.[$m][] | select(has("custom_fields"))] | length' "$OUT" 2> /dev/null)
    if [ -z "$n" ] || [ -z "$with" ]; then line FAIL "$no" "GET $route: no list in the member $member"
    elif [ "$n" -eq 0 ]; then line SKIP "$no" "GET $route: the list is empty, so the check says nothing"
    elif [ "$with" -eq "$n" ]; then line PASS "$no" "GET $route: all $n records have the member custom_fields"
    elif [ "$with" -eq 0 ]; then line FAIL "$no" "GET $route: none of the $n records has the member custom_fields. Every such statement would fail with X05"
    else line FAIL "$no" "GET $route: only $with of $n records have the member custom_fields. The server would show the others without custom values"; fi
}

check_12() {
    cf_list_check 12 "/products/$PID/features" features reference_num
    cf_list_check 12 /products products reference_prefix
}

check_17() {
    local ids id n
    [ "$MODE" = show ] && { req GET "/custom_field_definitions/{definition}/options"; return; }
    [ -s "$TMP/definitions.json" ] || { line SKIP 17 "no definitions were read"; return; }
    ids=$(jq -r '[.custom_field_definitions[]? | select(.type | test("SelectConstant$|SelectMultipleConstant$")) | .id] | .[0:5][]' "$TMP/definitions.json" 2> /dev/null)
    [ -n "$ids" ] || { line SKIP 17 "the account has no custom field with fixed options"; return; }
    for id in $ids; do
        req GET "/custom_field_definitions/$id/options"
        if ok && [ "$(j '.options | type')" = array ]; then
            n=$(j '[.options[] | select((.text | type) != "string")] | length')
            if [ "$n" -eq 0 ]; then line PASS 17 "options of the definition $id: $(j '[.options[].text] | join(" | ")')"
            else line FAIL 17 "options of the definition $id: $n options without the member text"; fi
        else
            line FAIL 17 "GET /custom_field_definitions/$id/options: status $STATUS: $(excerpt)"
        fi
    done
    line INFO 17 "compare these lists with the field settings in Aha!: are hidden options in them?"
}

# ---------------------------------------------------------------------------
# Write part
# ---------------------------------------------------------------------------

CREATED=()          # "route" entries for the deletion at the end, newest last

# create NAME ROUTE BODY MEMBER -> sets REF to the reference number (or id)
create() {
    local what=$1 route=$2 body=$3 member=$4
    REF=
    req POST "$route" "$body"
    [ "$MODE" = show ] && { REF="{$what}"; return 0; }
    if ok; then
        REF=$(jq -r --arg m "$member" '.[$m] | (.reference_num // .id) // empty' "$OUT")
    fi
    if [ -z "$REF" ]; then line FAIL setup "POST $route ($what): status $STATUS: $(excerpt)"; return 1; fi
    detail "created $what $REF"
    return 0
}

# same NO LABEL SENT GOT
same() {
    if [ "$3" = "$4" ]; then line PASS "$1" "$2: sent '$3', read back the same"
    else line FAIL "$1" "$2: sent '$3', read back '$4'"; fi
}

# cf_value KEY -> the JSON value of the entry of that custom field in $OUT (.feature)
cf_value() { jq -c --arg k "$1" '[.feature.custom_fields[]? | select(.key == $k)][0].value' "$OUT" 2> /dev/null; }

# pick_def TYPE-NAME -> the key of the first custom field of features of that type
pick_def() {
    jq -r --arg t "$1" '[.custom_field_definitions[]? | select((.custom_fieldable_type | split("::") | last) == "Feature") | select((.type | split("::") | last) == $t)][0].key // empty' "$TMP/definitions.json" 2> /dev/null
}

# pick_defs TYPE-NAME MAX -> one line "id<TAB>key" per custom field of
# features of that type, at most MAX lines
pick_defs() {
    jq -r --arg t "$1" --argjson max "$2" '[.custom_field_definitions[]? | select((.custom_fieldable_type | split("::") | last) == "Feature") | select((.type | split("::") | last) == $t)][0:$max][] | "\(.id)\t\(.key)"' "$TMP/definitions.json" 2> /dev/null
}

write_part() {
    local r1 r2 i1 f1 g1 g2 x1 c1 q1 e1 st2 gst qst est sid wt me_email me_id body got form route taken

    echo "Setup: records with the name prefix '$PREFIX' in the product $PRODUCT"
    req GET /me
    me_email=$(j '.user.email // empty'); me_id=$(j '.user.id // empty')
    [ "$MODE" = show ] && { me_email="{my e-mail}"; me_id="{my id}"; }
    req GET "/products/$PID/workflows"
    [ "$MODE" = show ] || cp "$OUT" "$TMP/workflows.json"
    gst=$(jq -r '[.workflows[]? | select(.statusable_type == "StrategicImperative" or .statusable_type == "Goal")][0].workflow_statuses[1].name // empty' "$TMP/workflows.json" 2> /dev/null)
    [ "$MODE" = show ] && gst="{second status of goals}"

    # Check 3: the description of a release goes into theme.
    body=$(jq -cn --arg n "$PREFIX R1" '{release: {name: $n, theme: "Line one\nLine two"}}')
    create release "/products/$PID/releases" "$body" release || return
    r1=$REF; CREATED+=("/products/$PID/releases/$r1")
    req GET "/releases/$r1"
    if [ "$MODE" != show ]; then
        got=$(j '.release.theme | if type == "object" then .body else . end')
        case "$got" in *"Line one"*"Line two"*) line PASS 3 "a release takes its description from theme" ;;
                       *) line FAIL 3 "release $r1: theme is '$got' after a create with theme" ;; esac
        detail "theme is stored as: $(j '.release.theme | if type == "object" then .body else . end | tojson')"
    fi
    body=$(jq -cn --arg n "$PREFIX R2" '{release: {name: $n}}')
    create "second release" "/products/$PID/releases" "$body" release || return
    r2=$REF; CREATED+=("/products/$PID/releases/$r2")
    body=$(jq -cn --arg n "$PREFIX I1" '{initiative: {name: $n}}')
    create initiative "/products/$PID/initiatives" "$body" initiative || return
    i1=$REF; CREATED+=("/products/$PID/initiatives/$i1")
    body=$(jq -cn --arg n "$PREFIX F1" '{feature: {name: $n, description: "Line one\nLine two"}}')
    create feature "/releases/$r1/features" "$body" feature || return
    f1=$REF; CREATED+=("/features/$f1")

    # The short run: only the checks 25 and 26 follow.
    if [ "${AHA_CHECK_SHORT:-}" = 1 ]; then
        PARAMS=("per_page=200" "page=1"); req GET /custom_field_definitions
        [ "$MODE" = show ] || cp "$OUT" "$TMP/definitions.json"
        check_25_26 "$f1" "$i1" "$me_email"
        return 0
    fi

    # Check 6: plain text with line breaks. Check 21: the workflow of a feature.
    req GET "/features/$f1"
    if [ "$MODE" = show ]; then
        st2="{another status of features}"
    else
        line INFO 6 "description of $f1 is stored as: $(j '.feature.description | if type == "object" then .body else . end | tojson')"
        sid=$(j '.feature.workflow_status.id // empty')
        wt=$(workflow_type_of "$TMP/workflows.json" "$sid")
        line INFO 21 "the status of a new feature is in a workflow with the record type: ${wt:-none of the list}"
        st2=$(other_status "$TMP/workflows.json" "$sid")
        [ -n "$st2" ] || st2=$(jq -r '[.workflows[]? | select(.statusable_type == "Feature")][0].workflow_statuses[1].name // empty' "$TMP/workflows.json" 2> /dev/null)
    fi
    body=$(jq -cn '{comment: {body: "Line one\nLine two"}}')
    if create comment "/features/$f1/comments" "$body" comment; then
        c1=$REF; CREATED+=("/comments/$c1")
        req GET "/comments/$c1"
        [ "$MODE" != show ] && line INFO 6 "body of the comment is stored as: $(j '.comment.body | tojson')"
    fi
    [ "$MODE" != show ] && line INFO 6 "by hand: see in Aha! whether the two lines of $f1 are shown as two lines. With AHA_CHECK_PAUSE=1 the script waits before it deletes"

    # Checks 2 and 7: an update with plain values, and its answer.
    if [ -n "$st2" ]; then
        body=$(jq -cn --arg s "$st2" --arg u "$me_email" --arg r "$r2" --arg i "$i1" --arg t "$TAGS" \
            '{feature: {workflow_status: $s, assigned_to_user: $u, release: $r, initiative: $i, tags: $t}}')
        req PUT "/features/$f1" "$body"
        if [ "$MODE" != show ]; then
            if ok; then
                if [ "$(j '.feature.workflow_status.name')" = "$st2" ] && [ "$(j '.feature.release_reference_num // .feature.release.reference_num')" = "$r2" ]; then
                    line PASS 7 "the answer to the update has the changed record"
                else
                    line FAIL 7 "the answer to the update has status '$(j '.feature.workflow_status.name')' and release '$(j '.feature.release_reference_num // .feature.release.reference_num')'. Members: $(j 'keys | join(", ")')"
                fi
                req GET "/features/$f1"
                same 2 workflow_status "$st2" "$(j '.feature.workflow_status.name')"
                same 2 assigned_to_user "$me_email" "$(j '.feature.assigned_to_user.email')"
                same 2 release "$r2" "$(j '.feature.release_reference_num // .feature.release.reference_num')"
                same 2 initiative "$i1" "$(j '.feature.initiative_reference_num // .feature.initiative.reference_num')"
                same 2 tags "$TAGS" "$(j '.feature.tags | sort | join(",")')"
            else
                line FAIL 2 "PUT /features/$f1 with plain values: status $STATUS: $(excerpt)"
            fi
        fi
    else
        line SKIP 2 "the workflow of features has no other status"
    fi

    # Check 21: the workflow of a requirement, and a status written to it.
    body=$(jq -cn --arg n "$PREFIX Q1" '{requirement: {name: $n}}')
    if create requirement "/features/$f1/requirements" "$body" requirement; then
        q1=$REF; CREATED+=("/requirements/$q1")
        req GET "/requirements/$q1"
        if [ "$MODE" = show ]; then
            qst="{another status of requirements}"
        else
            sid=$(j '.requirement.workflow_status.id // empty')
            wt=$(workflow_type_of "$TMP/workflows.json" "$sid")
            line INFO 21 "the status of a new requirement is in a workflow with the record type: ${wt:-none of the list}"
            qst=$(other_status "$TMP/workflows.json" "$sid")
        fi
        if [ -n "$qst" ]; then
            body=$(jq -cn --arg s "$qst" '{requirement: {workflow_status: $s}}')
            req PUT "/requirements/$q1" "$body"
            if [ "$MODE" != show ]; then
                if ok; then
                    req GET "/requirements/$q1"
                    same 21 "status of a requirement" "$qst" "$(j '.requirement.workflow_status.name')"
                else
                    line FAIL 21 "PUT /requirements/$q1 with workflow_status: status $STATUS: $(excerpt)"
                fi
            fi
        else
            line SKIP 21 "no other status was found for the requirement"
        fi
    fi

    # Checks 21 and 22: an epic, where the test product takes one.
    body=$(jq -cn --arg n "$PREFIX E1" '{epic: {name: $n}}')
    req POST "/releases/$r1/epics" "$body"
    if [ "$MODE" = show ]; then
        req PUT "/epics/{epic}" '{"epic":{"workflow_status":"{another status of epics}"}}'
        CREATED+=("/epics/{epic}")
    else
        e1=$(j '.epic.reference_num // empty')
        if ok && [ -n "$e1" ]; then
            CREATED+=("/epics/$e1"); detail "created epic $e1"
            sid=$(j '.epic.workflow_status.id // empty')
            wt=$(workflow_type_of "$TMP/workflows.json" "$sid")
            line INFO 21 "the status of a new epic is in a workflow with the record type: ${wt:-none of the list}"
            line INFO 22 "the new epic: release.reference_num is $(j '.epic.release.reference_num | type'), the member release_reference_num is $(j 'if (.epic | has("release_reference_num")) then "there" else "absent" end'), the member color is $(j 'if (.epic | has("color")) then "there" else "absent" end')"
            est=$(other_status "$TMP/workflows.json" "$sid")
            if [ -n "$est" ]; then
                body=$(jq -cn --arg s "$est" '{epic: {workflow_status: $s}}')
                req PUT "/epics/$e1" "$body"
                if ok; then
                    req GET "/epics/$e1"
                    same 21 "status of an epic" "$est" "$(j '.epic.workflow_status.name')"
                else
                    line FAIL 21 "PUT /epics/$e1 with workflow_status: status $STATUS: $(excerpt)"
                fi
            fi
        else
            line INFO 22 "no epic was created in the test product: status $STATUS: $(excerpt)"
        fi
    fi

    # Checks 4, 23 and 24: a goal, its status and its success metric.
    # Aha! creates no goal without a success metric that has a name (runs of
    # 2026-11-24). The bodies below are those of write-request.
    g1=; g2=
    body=$(jq -cn --arg n "$PREFIX G1" '{goal: {name: $n, success_metric: {name: "aql-check"}}}')
    if create goal "/products/$PID/goals" "$body" goal; then
        g1=$REF; CREATED+=("/products/$PID/goals/$g1")
        [ "$MODE" != show ] && line PASS 4 "a goal is created with a success metric that has a name"
        if [ -n "$gst" ]; then
            # Check 4: an update takes the status as workflow_status.
            body=$(jq -cn --arg s "$gst" '{goal: {workflow_status: $s}}')
            req PUT "/goals/$g1" "$body"
            if [ "$MODE" != show ]; then
                if ok; then
                    req GET "/goals/$g1"
                    same 4 "status of a goal in an update, sent as workflow_status" "$gst" "$(j '.goal.success_metric.workflow_status.name // .goal.workflow_status.name')"
                else
                    line FAIL 4 "PUT /goals/$g1 with workflow_status: status $STATUS: $(excerpt)"
                fi
            fi
            # Check 24: the text of the success metric in an update. The first
            # run of this check showed that Aha! answers the first form
            # without an error and keeps the old text. Since version 0.13.0 the server
            # therefore sends no such update. The forms below look for one that
            # Aha! takes. A plain text as the value of success_metric is not
            # among them: Aha! answered it with the status 500 on 2026-11-24.
            if [ "$MODE" = show ]; then
                req PUT "/goals/$g1" '{"goal":{"success_metric":{"name":"aql-check a"}}}'
                req PUT "/products/$PID/goals/$g1" '{"goal":{"success_metric":{"name":"aql-check b"}}}'
                req PUT "/goals/$g1" '{"goal":{"success_metric":{"name":"aql-check c","workflow_status":"{second status of goals}"}}}'
            else
                taken=
                for form in a b c; do
                    route="/goals/$g1"
                    case "$form" in
                        a) body='{"goal":{"success_metric":{"name":"aql-check a"}}}' ;;
                        b) body='{"goal":{"success_metric":{"name":"aql-check b"}}}'; route="/products/$PID/goals/$g1" ;;
                        c) body=$(jq -cn --arg s "$gst" '{goal: {success_metric: {name: "aql-check c", workflow_status: $s}}}') ;;
                    esac
                    req PUT "$route" "$body"
                    if ok; then
                        req GET "/goals/$g1"
                        got=$(j '.goal.success_metric.name')
                        if [ "$got" = "aql-check $form" ]; then
                            taken=1
                            line NOTE 24 "Aha! takes a new text of the success metric with PUT $route and the body $body. The status is then '$(j '.goal.success_metric.workflow_status.name')'. The server does not offer this update"
                        else
                            detail "PUT $route $body: status 200, and the text is still '$got'"
                        fi
                    else
                        detail "PUT $route $body: status $STATUS: $(excerpt)"
                    fi
                done
                [ -n "$taken" ] || line PASS 24 "no tried form changes the text of the success metric in an update. The server refuses such an UPDATE"
            fi
            # Check 23: a new goal takes its status inside success_metric.
            body=$(jq -cn --arg n "$PREFIX G2" --arg s "$gst" '{goal: {name: $n, success_metric: {name: "aql-check", workflow_status: $s}}}')
            if create "second goal" "/products/$PID/goals" "$body" goal; then
                g2=$REF; CREATED+=("/products/$PID/goals/$g2")
                req GET "/goals/$g2"
                [ "$MODE" != show ] && same 23 "status of a new goal, sent inside success_metric" "$gst" "$(j '.goal.success_metric.workflow_status.name')"
            else
                line FAIL 23 "see the line above: Aha! refuses a new goal with its status inside success_metric"
            fi
        else
            line SKIP 4 "no workflow for goals with a second status was found, so the checks 4, 23 and 24 of the status did not run"
        fi
    else
        line FAIL 4 "see the line above: no goal was created"
    fi

    # Check 5: an idea is created with POST.
    body=$(jq -cn --arg n "$PREFIX X1" '{idea: {name: $n}}')
    if create idea "/products/$PID/ideas" "$body" idea; then
        x1=$REF; CREATED+=("/ideas/$x1")
        [ "$MODE" != show ] && line PASS 5 "an idea can be created with POST /products/{product}/ideas: $x1"
    else
        line FAIL 5 "see the line above. A product without ideas answers so, too"
    fi

    # Checks 13 to 16 and 18: custom fields of the feature.
    PARAMS=("per_page=200" "page=1"); req GET /custom_field_definitions
    if [ "$MODE" = show ]; then
        req PUT "/features/$f1" '{"feature":{"name":"...","custom_fields":{"{text key}":"text value"}}}'
        req PUT "/features/$f1" '{"feature":{"custom_fields":{"{number key}":12.5}}}'
        req PUT "/features/$f1" '{"feature":{"custom_fields":{"{note key}":"Line one\nLine two"}}}'
        req GET "/custom_field_definitions/{fixed choice}/options"
        req PUT "/features/$f1" '{"feature":{"custom_fields":{"{fixed choice key}":"{text of its first option}"}}}'
        req PUT "/features/$f1" '{"feature":{"custom_fields":{"{fixed tags key}":["{first option}","{second option}"]}}}'
        req PUT "/features/$f1" '{"feature":{"custom_fields":{"{users key}":["{my id}"]}}}'
        req PUT "/features/$f1" '{"feature":{"custom_fields":{"{users key}":["{my e-mail}"]}}}'
        req PUT "/features/$f1" '{"feature":{"custom_fields":{"{text key}":"","{fixed tags key}":[],"{users key}":[]}}}'
        echo "          only with AHA_CHECK_EDITABLE=1:"
        req PUT "/features/$f1" '{"feature":{"custom_fields":{"{tags key}":["aql-check-1","aql-check-2"]}}}'
    else
        cp "$OUT" "$TMP/definitions.json"
        cf_checks "$f1" "$me_email" "$me_id"
    fi

    # Checks 25 and 26: values that are removed, and records that do not exist.
    check_25_26 "$f1" "$i1" "$me_email"
}

# put_cf NO FEATURE JSON-BODY -> 0 when the update and the read back succeeded
put_cf() {
    req PUT "/features/$2" "$3"
    if ! ok; then line FAIL "$1" "PUT /features/$2 $3: status $STATUS: $(excerpt)"; return 1; fi
    req GET "/features/$2"
    ok
}

cf_checks() {
    local f1=$1 me_email=$2 me_id=$3 id k kt kn ktags kno ku body opt opts last_fixed_tags=
    kt=$(pick_def TextField); kn=$(pick_def NumberField); ktags=$(pick_def SelectMultipleEditable)
    kno=$(pick_def NoteField); ku=$(pick_def UsersField)
    pick_defs SelectConstant 5 > "$TMP/fixed-choice.tsv"
    pick_defs SelectMultipleConstant 5 > "$TMP/fixed-tags.tsv"
    detail "custom fields of features used: text '${kt:-none}', number '${kn:-none}', tags '${ktags:-none}', note '${kno:-none}', users '${ku:-none}',"
    detail "fixed choice: $(cut -f2 "$TMP/fixed-choice.tsv" | paste -sd' ' -), fixed tags: $(cut -f2 "$TMP/fixed-tags.tsv" | paste -sd' ' -)"

    if [ -n "$kt" ]; then
        body=$(jq -cn --arg n "$PREFIX F1 b" --arg k "$kt" '{feature: {name: $n, custom_fields: {($k): "text value"}}}')
        if put_cf 13 "$f1" "$body"; then
            if [ "$(j '.feature.name')" = "$PREFIX F1 b" ] && [ "$(cf_value "$kt")" = '"text value"' ]; then
                line PASS 13 "an update with custom_fields next to name changes both"
            else
                line FAIL 13 "after the update: name '$(j '.feature.name')', $kt $(cf_value "$kt")"
            fi
        fi
    else
        line SKIP 13 "features have no custom field of the type TextField"
    fi

    if [ -n "$kn" ]; then
        body=$(jq -cn --arg k "$kn" '{feature: {custom_fields: {($k): 12.5}}}')
        put_cf 14 "$f1" "$body" && line INFO 14 "number sent as 12.5, stored as $(cf_value "$kn")"
    else line SKIP 14 "features have no custom field of the type NumberField"; fi
    if [ -n "$kno" ]; then
        body=$(jq -cn --arg k "$kno" '{feature: {custom_fields: {($k): "Line one\nLine two"}}}')
        put_cf 14 "$f1" "$body" && line INFO 14 "note sent as plain text with a line break, stored as $(cf_value "$kno")"
    else line SKIP 14 "features have no custom field of the type NoteField"; fi

    # Lists with fixed options: the value is the text of an option. A record
    # cannot add an option to such a list.
    if [ -s "$TMP/fixed-choice.tsv" ]; then
        while IFS=$'\t' read -r id k <&3; do
            req GET "/custom_field_definitions/$id/options"
            opt=$(j '.options[0].text // empty')
            if [ -z "$opt" ]; then line SKIP 14 "fixed choice '$k': no option was read (status $STATUS)"; continue; fi
            body=$(jq -cn --arg k "$k" --arg v "$opt" '{feature: {custom_fields: {($k): $v}}}')
            put_cf 14 "$f1" "$body" && line INFO 14 "fixed choice '$k' sent as the text of its first option, stored as $(cf_value "$k")"
        done 3< "$TMP/fixed-choice.tsv"
    else
        line SKIP 14 "features have no custom field of the type SelectConstant"
    fi
    if [ -s "$TMP/fixed-tags.tsv" ]; then
        while IFS=$'\t' read -r id k <&3; do
            req GET "/custom_field_definitions/$id/options"
            opts=$(j -c '[.options[]?.text][0:2]')
            if [ -z "$opts" ] || [ "$opts" = "[]" ]; then line SKIP 14 "fixed tags '$k': no option was read (status $STATUS)"; continue; fi
            body=$(jq -cn --arg k "$k" --argjson v "$opts" '{feature: {custom_fields: {($k): $v}}}')
            put_cf 14 "$f1" "$body" && line INFO 14 "fixed tags '$k' sent as $opts, stored as $(cf_value "$k")"
            last_fixed_tags=$k
        done 3< "$TMP/fixed-tags.tsv"
    else
        line SKIP 14 "features have no custom field of the type SelectMultipleConstant"
    fi
    line INFO 14 "'null' above means that Aha! did not store the value. The field is then outside the layout of the test product (check 18), or Aha! wants another form"

    # A list that takes new values can keep them as choices in every
    # product. So this runs only on request.
    if [ -z "$ktags" ]; then
        line SKIP 14 "features have no custom field of the type SelectMultipleEditable"
    elif [ "${AHA_CHECK_EDITABLE:-}" = 1 ]; then
        body=$(jq -cn --arg k "$ktags" '{feature: {custom_fields: {($k): ["aql-check-1", "aql-check-2"]}}}')
        put_cf 14 "$f1" "$body" && line INFO 14 "tags sent as [\"aql-check-1\",\"aql-check-2\"], stored as $(cf_value "$ktags")"
    else
        line SKIP 14 "tags field '$ktags': Aha! can keep new tags as choices of that field in every product. Set AHA_CHECK_EDITABLE=1 to run this"
        ktags=
    fi

    if [ -n "$ku" ]; then
        body=$(jq -cn --arg k "$ku" --arg id "$me_id" '{feature: {custom_fields: {($k): [$id]}}}')
        put_cf 15 "$f1" "$body" && line INFO 15 "users sent as a list with one user id, stored as $(cf_value "$ku")"
        body=$(jq -cn --arg k "$ku" '{feature: {custom_fields: {($k): []}}}')
        put_cf 16 "$f1" "$body" && line INFO 16 "users sent as an empty list, stored as $(cf_value "$ku")"
        body=$(jq -cn --arg k "$ku" --arg m "$me_email" '{feature: {custom_fields: {($k): [$m]}}}')
        req PUT "/features/$f1" "$body"
        if ok; then
            req GET "/features/$f1"
            line INFO 15 "users sent as a list with one e-mail address: status 200, stored as $(cf_value "$ku")"
        else
            line INFO 15 "users sent as a list with one e-mail address: status $STATUS: $(excerpt)"
        fi
    else
        line SKIP 15 "features have no custom field of the type UsersField"
    fi

    if [ -n "$kt" ]; then
        body=$(jq -cn --arg k "$kt" '{feature: {custom_fields: {($k): ""}}}')
        put_cf 16 "$f1" "$body" && line INFO 16 "text sent as empty text, stored as $(cf_value "$kt")"
    fi
    if [ -n "$ktags" ]; then
        body=$(jq -cn --arg k "$ktags" '{feature: {custom_fields: {($k): []}}}')
        put_cf 16 "$f1" "$body" && line INFO 16 "tags sent as an empty list, stored as $(cf_value "$ktags")"
    fi
    if [ -n "$last_fixed_tags" ]; then
        body=$(jq -cn --arg k "$last_fixed_tags" '{feature: {custom_fields: {($k): []}}}')
        put_cf 16 "$f1" "$body" && line INFO 16 "fixed tags '$last_fixed_tags' sent as an empty list, stored as $(cf_value "$last_fixed_tags")"
    fi
    line INFO 16 "'null' or an empty value above means that the value is removed"

    k=${AHA_CHECK_FIELD_NOT_IN_LAYOUT:-}
    if [ -n "$k" ]; then
        body=$(jq -cn --arg k "$k" '{feature: {custom_fields: {($k): "outside the layout"}}}')
        req PUT "/features/$f1" "$body"
        if ok; then
            req GET "/features/$f1"
            line INFO 18 "a field outside the layout: status 200, stored as $(cf_value "$k")"
        else
            line INFO 18 "a field outside the layout: status $STATUS: $(excerpt)"
        fi
    else
        line SKIP 18 "set AHA_CHECK_FIELD_NOT_IN_LAYOUT to the key of such a field"
    fi
}

# ---------------------------------------------------------------------------
# Check 25: which request removes a value: empty text or the JSON null
# Check 26: an update that names a release or an initiative that does not
# exist
# ---------------------------------------------------------------------------

# gone JSON-VALUE: the field has no value
gone() { case "$1" in null|'""'|'[]') return 0 ;; *) return 1 ;; esac; }

# read_value FEATURE JQ-PROGRAM [KEY] -> sets VAL to the value of one field
# of the feature, as JSON. The program gets KEY as $k. Returns 1 when the
# feature cannot be read.
read_value() {
    VAL=
    req GET "/features/$1"
    ok || return 1
    VAL=$(jq -c --arg k "${3-}" "$2" "$OUT" 2> /dev/null)
    [ -n "$VAL" ] || VAL="(not readable)"
    return 0
}

# removes LABEL FEATURE SET-BODY TEXT-BODY NULL-BODY JQ-PROGRAM [KEY]
# Sets a value. Then it sends empty text and, when the value is still
# there, the JSON null. The lines say which request removes the value.
removes() {
    local label=$1 f=$2 set=$3 text=$4 null=$5 prog=$6 key=${7-} before st_text st_null
    req PUT "/features/$f" "$set"
    if [ "$MODE" = show ]; then
        req PUT "/features/$f" "$text"; req PUT "/features/$f" "$null"; return 0
    fi
    if ! ok; then line SKIP 25 "$label: the value could not be set first: status $STATUS: $(excerpt)"; return 0; fi
    if ! read_value "$f" "$prog" "$key"; then line FAIL 25 "$label: GET /features/$f: status $STATUS: $(excerpt)"; return 0; fi
    before=$VAL
    if gone "$before"; then line SKIP 25 "$label: Aha! did not store the value of the first request, so there is nothing to remove"; return 0; fi
    req PUT "/features/$f" "$text"; st_text=$STATUS
    if ! read_value "$f" "$prog" "$key"; then line FAIL 25 "$label: GET /features/$f: status $STATUS: $(excerpt)"; return 0; fi
    case "$st_text" in 2??)
        if gone "$VAL"; then line INFO 25 "$label: empty text removes the value (status $st_text). Before: $before"; return 0; fi ;;
    esac
    detail "$label sent as empty text: status $st_text, and the value is then $VAL"
    req PUT "/features/$f" "$null"; st_null=$STATUS
    if ! read_value "$f" "$prog" "$key"; then line FAIL 25 "$label: GET /features/$f: status $STATUS: $(excerpt)"; return 0; fi
    case "$st_null" in 2??)
        if gone "$VAL"; then line INFO 25 "$label: the JSON null removes the value (status $st_null). Empty text does not (status $st_text). Before: $before"; return 0; fi ;;
    esac
    line INFO 25 "$label: empty text (status $st_text) and the JSON null (status $st_null) do not remove the value. It is $VAL"
}

# missing_record LABEL FEATURE MEMBER VALUE JQ-PROGRAM
# Sends an update that names a record which does not exist.
missing_record() {
    local label=$1 f=$2 member=$3 value=$4 prog=$5 body before st answer
    body=$(jq -cn --arg m "$member" --arg v "$value" '{feature: {($m): $v}}')
    if [ "$MODE" = show ]; then req PUT "/features/$f" "$body"; return 0; fi
    if ! read_value "$f" "$prog"; then line FAIL 26 "$label: GET /features/$f: status $STATUS: $(excerpt)"; return 0; fi
    before=$VAL
    req PUT "/features/$f" "$body"; st=$STATUS; answer=$(excerpt)
    if ! read_value "$f" "$prog"; then line FAIL 26 "$label: GET /features/$f: status $STATUS: $(excerpt)"; return 0; fi
    case "$st" in
        404) line PASS 26 "an update with $label that does not exist: status 404, as update-statement expects. The value is still $VAL" ;;
        2??) if [ "$VAL" = "$before" ]; then
                 line NOTE 26 "an update with $label that does not exist: status $st and no error. The value is still $before"
             else
                 line NOTE 26 "an update with $label that does not exist: status $st, and the value changed from $before to $VAL"
             fi ;;
        *)   line INFO 26 "an update with $label that does not exist: status $st: $answer. The value is then $VAL" ;;
    esac
}

# check_25_26 FEATURE INITIATIVE MY-E-MAIL
# The feature is the one of the setup. The values that the checks set stay
# on that feature, and the script deletes it at the end.
# shellcheck disable=SC2016  # jq programs, not shell
check_25_26() {
    local f=$1 i1=$2 me_email=$3 kd kn set text null
    local p_due='.feature.due_date'
    local p_user='.feature.assigned_to_user | if type == "object" then .email else . end'
    local p_init='.feature.initiative_reference_num // .feature.initiative.reference_num'
    local p_rel='.feature.release_reference_num // .feature.release.reference_num'
    local p_cf='[.feature.custom_fields[]? | select(.key == $k)][0].value'

    missing_record "a release" "$f" release "$PRODUCT-R-99999999" "$p_rel"
    missing_record "an initiative" "$f" initiative "$PRODUCT-S-99999999" "$p_init"

    removes due_date "$f" '{"feature":{"due_date":"2026-12-31"}}' \
        '{"feature":{"due_date":""}}' '{"feature":{"due_date":null}}' "$p_due"
    set=$(jq -cn --arg u "$me_email" '{feature: {assigned_to_user: $u}}')
    removes assigned_to_user "$f" "$set" \
        '{"feature":{"assigned_to_user":""}}' '{"feature":{"assigned_to_user":null}}' "$p_user"
    set=$(jq -cn --arg i "$i1" '{feature: {initiative: $i}}')
    removes initiative "$f" "$set" \
        '{"feature":{"initiative":""}}' '{"feature":{"initiative":null}}' "$p_init"

    if [ "$MODE" = show ]; then kd="{date key}"; kn="{number key}"
    else kd=$(pick_def DateField); kn=$(pick_def NumberField); fi
    if [ -n "$kd" ]; then
        set=$(jq -cn --arg k "$kd" '{feature: {custom_fields: {($k): "2026-12-31"}}}')
        text=$(jq -cn --arg k "$kd" '{feature: {custom_fields: {($k): ""}}}')
        null=$(jq -cn --arg k "$kd" '{feature: {custom_fields: {($k): null}}}')
        removes "custom date '$kd'" "$f" "$set" "$text" "$null" "$p_cf" "$kd"
    else
        line SKIP 25 "features have no custom field of the type DateField"
    fi
    if [ -n "$kn" ]; then
        set=$(jq -cn --arg k "$kn" '{feature: {custom_fields: {($k): 12.5}}}')
        text=$(jq -cn --arg k "$kn" '{feature: {custom_fields: {($k): ""}}}')
        null=$(jq -cn --arg k "$kn" '{feature: {custom_fields: {($k): null}}}')
        removes "custom number '$kn'" "$f" "$set" "$text" "$null" "$p_cf" "$kn"
    else
        line SKIP 25 "features have no custom field of the type NumberField"
    fi
}

cleanup_records() {
    local i route
    [ ${#CREATED[@]} -gt 0 ] || return 0
    if [ "${AHA_CHECK_KEEP:-}" = 1 ]; then
        echo "Kept, because AHA_CHECK_KEEP is 1:"; printf '  %s\n' "${CREATED[@]}"; return 0
    fi
    if [ "${AHA_CHECK_PAUSE:-}" = 1 ] && [ "$MODE" = write ]; then
        printf 'The records are still there. Look at them in Aha!, then press Enter to delete them: '
        read -r _ < /dev/tty
    fi
    echo "Deleting the records of this run, newest first"
    for ((i = ${#CREATED[@]} - 1; i >= 0; i--)); do
        route=${CREATED[$i]}
        req DELETE "$route"
        if [ "$MODE" != show ]; then
            if ok; then detail "deleted $route"; else line FAIL clean "DELETE $route: status $STATUS: $(excerpt)"; fi
        fi
    done
}

# ---------------------------------------------------------------------------

summary() {
    echo "Summary: ${COUNT[PASS]} PASS, ${COUNT[FAIL]} FAIL, ${COUNT[NOTE]} NOTE, ${COUNT[INFO]} INFO, ${COUNT[SKIP]} SKIP"
    [ "${COUNT[NOTE]}" -eq 0 ] || echo "A NOTE line: the request works, and the answer differs from what version 0.20.0 expects."
    [ "${COUNT[FAIL]}" -eq 0 ]
}

if [ "$MODE" = products ]; then
    echo "aha-checks products: account $SUB, $(date -u +%Y-%m-%dT%H:%M:%SZ)"
    load_products
    if [ -n "$FILTER" ]; then
        echo "$NPRODUCTS products are visible with this API key. With '$FILTER' in the key or in the name:"
    else
        echo "$NPRODUCTS products are visible with this API key. Key and name:"
    fi
    LISTED=$(match_products part "$FILTER")
    if [ -n "$LISTED" ]; then printf '%s\n' "$LISTED" | print_products; else echo "  none"; fi
    exit 0
fi

echo "aha-checks $MODE: account $SUB, test product $PRODUCT, $(date -u +%Y-%m-%dT%H:%M:%SZ)"
if [ "$MODE" = show ]; then
    echo "The requests of the read part:"
    find_product; check_1; check_forms; check_21; check_22; check_8; check_10_11; check_12; check_17
    echo "The requests of the write part:"
    write_part; cleanup_records
    exit 0
fi

find_product
if [ -z "$NOID" ]; then IDTEXT="id $PID"; else IDTEXT="no id in the list"; fi
echo "The test product: key $PRODUCT, name '$PNAME', $IDTEXT. The account shows $NPRODUCTS products to this API key"
[ -z "$NOID" ] || line FAIL 1 "GET /products: the records of the list have no member id. The specification expects it in every record of a list. The routes get the key from here on"

if [ "$MODE" = read ]; then
    check_1
    check_forms
    check_21
    check_22
    check_8
    check_10_11
    check_12
    check_17
    line INFO 9 "by hand: read a record of a product that the user of the key cannot see. The server expects the status 404"
else
    if [ "${AHA_CHECK_SHORT:-}" = 1 ]; then
        cat <<TEXT
The short write part creates these records in the product $PRODUCT, '$PNAME', of the account $SUB:
2 releases, 1 initiative and 1 feature. It deletes them at the end.
TEXT
    else
        cat <<TEXT
The write part creates these records in the product $PRODUCT, '$PNAME', of the account $SUB:
2 releases, 1 initiative, 2 goals, 1 idea, 1 feature with 1 comment and 1 requirement,
and 1 epic where the product takes one. It deletes them at the end.
The tags ${TAGS/,/ and } of its feature can stay in the list of tags of the account.
TEXT
    fi
    printf 'Type the product key to go on: '
    read -r answer
    [ "${answer^^}" = "${PRODUCT^^}" ] || { echo "stopped"; exit 1; }
    trap 'echo; echo "interrupted"; AHA_CHECK_PAUSE=0; cleanup_records; exit 130' INT TERM
    write_part
    cleanup_records
    trap - INT TERM
fi
summary
