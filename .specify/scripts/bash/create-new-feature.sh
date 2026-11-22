#!/usr/bin/env bash

set -e

JSON_MODE=false
DRY_RUN=false
ALLOW_EXISTING=false
SHORT_NAME=""
BRANCH_NUMBER=""
USE_TIMESTAMP=false
BRANCH_TYPE="feat"
BASE_BRANCH="develop"
EXPLICIT_BASE=false
CREATE_BRANCH=false
ARGS=()
i=1
while [ $i -le $# ]; do
    arg="${!i}"
    case "$arg" in
        --json)
            JSON_MODE=true
            ;;
        --dry-run)
            DRY_RUN=true
            ;;
        --allow-existing-branch)
            ALLOW_EXISTING=true
            ;;
        --short-name)
            if [ $((i + 1)) -gt $# ]; then
                echo 'Error: --short-name requires a value' >&2
                exit 1
            fi
            i=$((i + 1))
            next_arg="${!i}"
            # Check if the next argument is another option (starts with --)
            if [[ "$next_arg" == --* ]]; then
                echo 'Error: --short-name requires a value' >&2
                exit 1
            fi
            SHORT_NAME="$next_arg"
            ;;
        --number)
            if [ $((i + 1)) -gt $# ]; then
                echo 'Error: --number requires a value' >&2
                exit 1
            fi
            i=$((i + 1))
            next_arg="${!i}"
            if [[ "$next_arg" == --* ]]; then
                echo 'Error: --number requires a value' >&2
                exit 1
            fi
            BRANCH_NUMBER="$next_arg"
            ;;
        --branch-type)
            if [ $((i + 1)) -gt $# ]; then
                echo 'Error: --branch-type requires a value' >&2
                exit 1
            fi
            i=$((i + 1))
            next_arg="${!i}"
            if [[ "$next_arg" == --* ]]; then
                echo 'Error: --branch-type requires a value' >&2
                exit 1
            fi
            BRANCH_TYPE="$next_arg"
            ;;
        --base)
            if [ $((i + 1)) -gt $# ]; then
                echo 'Error: --base requires a value' >&2
                exit 1
            fi
            i=$((i + 1))
            next_arg="${!i}"
            if [[ "$next_arg" == --* ]]; then
                echo 'Error: --base requires a value' >&2
                exit 1
            fi
            BASE_BRANCH="$next_arg"
            EXPLICIT_BASE=true
            ;;
        --create-branch)
            CREATE_BRANCH=true
            ;;
        --timestamp)
            USE_TIMESTAMP=true
            ;;
        --help|-h)
            echo "Usage: $0 [--json] [--dry-run] [--allow-existing-branch] [--short-name <name>] [--number N] [--timestamp] [--branch-type <type>] [--base <branch>] [--create-branch] <feature_description>"
            echo ""
            echo "Options:"
            echo "  --json                  Output in JSON format"
            echo "  --dry-run               Compute feature name and paths without creating directories, files, or branches"
            echo "  --allow-existing-branch Reuse an existing feature directory or git branch if it already exists"
            echo "  --short-name <name>     Provide a custom short name (2-4 words) for the feature"
            echo "  --number N              Specify branch/spec number manually (overrides auto-detection)"
            echo "  --timestamp             Use timestamp prefix (YYYYMMDD-HHMMSS) instead of sequential numbering"
            echo "  --branch-type <type>    Git Flow branch prefix: feat, feature, fix, docs, chore, ci, refactor, test (default: feat)"
            echo "  --base <branch>         Base branch for Git Flow branch creation (default: develop)"
            echo "  --create-branch         Create and switch to the Git Flow branch (<type>/<short-name>) from --base"
            echo "  --help, -h              Show this help message"
            echo ""
            echo "Examples:"
            echo "  $0 'Add user authentication system' --short-name 'user-auth'"
            echo "  $0 --create-branch --short-name 'user-auth' 'Add user authentication system'"
            echo "  $0 --branch-type fix --create-branch --short-name 'yaml-bug' 'Fix YAML indent bug'"
            echo "  $0 'Implement OAuth2 integration for API' --number 5"
            echo "  $0 --timestamp --short-name 'user-auth' 'Add user authentication'"
            exit 0
            ;;
        *)
            ARGS+=("$arg")
            ;;
    esac
    i=$((i + 1))
done

case "$BRANCH_TYPE" in
    feat|feature|fix|docs|chore|ci|refactor|test)
        ;;
    *)
        echo "Error: Invalid --branch-type '$BRANCH_TYPE'. Allowed values: feat, feature, fix, docs, chore, ci, refactor, test" >&2
        exit 1
        ;;
esac

FEATURE_DESCRIPTION="${ARGS[*]}"
if [ -z "$FEATURE_DESCRIPTION" ]; then
    echo "Usage: $0 [--json] [--dry-run] [--allow-existing-branch] [--short-name <name>] [--number N] [--timestamp] [--branch-type <type>] [--base <branch>] [--create-branch] <feature_description>" >&2
    exit 1
fi

# Trim whitespace and validate description is not empty (e.g., user passed only whitespace)
FEATURE_DESCRIPTION=$(echo "$FEATURE_DESCRIPTION" | sed -E 's/^[[:space:]]+|[[:space:]]+$//g')
if [ -z "$FEATURE_DESCRIPTION" ]; then
    echo "Error: Feature description cannot be empty or contain only whitespace" >&2
    exit 1
fi

# Function to get highest number from specs directory
get_highest_from_specs() {
    local specs_dir="$1"
    local highest=0

    if [ -d "$specs_dir" ]; then
        for dir in "$specs_dir"/*; do
            [ -d "$dir" ] || continue
            dirname=$(basename "$dir")
            # Match sequential prefixes (>=3 digits), but skip timestamp dirs.
            if echo "$dirname" | grep -Eq '^[0-9]{3,}-' && ! echo "$dirname" | grep -Eq '^[0-9]{8}-[0-9]{6}-'; then
                number=$(echo "$dirname" | grep -Eo '^[0-9]+')
                number=$((10#$number))
                if [ "$number" -gt "$highest" ]; then
                    highest=$number
                fi
            fi
        done
    fi

    echo "$highest"
}

# Function to clean and format a branch name
clean_branch_name() {
    local name="$1"
    echo "$name" | tr '[:upper:]' '[:lower:]' | sed 's/[^a-z0-9]/-/g' | sed 's/-\+/-/g' | sed 's/^-//' | sed 's/-$//'
}

# Resolve repository root using common.sh functions which prioritize .specify
SCRIPT_DIR="$(CDPATH="" cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/common.sh"

REPO_ROOT=$(get_repo_root) || exit 1

cd "$REPO_ROOT"

SPECS_DIR="$REPO_ROOT/specs"

# Function to generate branch name with stop word filtering and length filtering
generate_branch_name() {
    local description="$1"

    # Common stop words to filter out
    local stop_words="^(i|a|an|the|to|for|of|in|on|at|by|with|from|is|are|was|were|be|been|being|have|has|had|do|does|did|will|would|should|could|can|may|might|must|shall|this|that|these|those|my|your|our|their|want|need|add|get|set)$"

    # Convert to lowercase and split into words
    local clean_name=$(printf '%s' "$description" | tr '[:upper:]' '[:lower:]' | sed 's/[^a-z0-9]/ /g')

    # Filter words: remove stop words and words shorter than 3 chars (unless they're uppercase acronyms in original)
    local meaningful_words=()
    for word in $clean_name; do
        # Skip empty words
        [ -z "$word" ] && continue

        # Keep words that are NOT stop words AND (length >= 3 OR are potential acronyms)
        if ! echo "$word" | grep -qiE "$stop_words"; then
            if [ ${#word} -ge 3 ]; then
                meaningful_words+=("$word")
            # Keep short words that appear as an uppercase acronym in the original.
            # Uppercase via tr and match with grep -w (both portable) rather than
            # bash's 4+ "^^" case expansion (breaks on macOS bash 3.2) and \b (non-POSIX).
            elif printf '%s' "$description" | grep -qw -- "$(printf '%s' "$word" | tr '[:lower:]' '[:upper:]')"; then
                meaningful_words+=("$word")
            fi
        fi
    done

    # If we have meaningful words, use first 3-4 of them
    if [ ${#meaningful_words[@]} -gt 0 ]; then
        local max_words=3
        if [ ${#meaningful_words[@]} -eq 4 ]; then max_words=4; fi

        local result=""
        local count=0
        for word in "${meaningful_words[@]}"; do
            if [ $count -ge $max_words ]; then break; fi
            if [ -n "$result" ]; then result="$result-"; fi
            result="$result$word"
            count=$((count + 1))
        done
        echo "$result"
    else
        # Fallback to original logic if no meaningful words found
        local cleaned=$(clean_branch_name "$description")
        echo "$cleaned" | tr '-' '\n' | grep -v '^$' | head -3 | tr '\n' '-' | sed 's/-$//'
    fi
}

# Generate branch name
if [ -n "$SHORT_NAME" ]; then
    # Use provided short name, just clean it up
    BRANCH_SUFFIX=$(clean_branch_name "$SHORT_NAME")
else
    # Generate from description with smart filtering
    BRANCH_SUFFIX=$(generate_branch_name "$FEATURE_DESCRIPTION")
fi

# Warn if --number and --timestamp are both specified
if [ "$USE_TIMESTAMP" = true ] && [ -n "$BRANCH_NUMBER" ]; then
    >&2 echo "[specify] Warning: --number is ignored when --timestamp is used"
    BRANCH_NUMBER=""
fi

# Determine branch prefix
if [ "$USE_TIMESTAMP" = true ]; then
    FEATURE_NUM=$(date +%Y%m%d-%H%M%S)
    BRANCH_NAME="${FEATURE_NUM}-${BRANCH_SUFFIX}"
else
    # Determine branch number from existing feature directories
    if [ -z "$BRANCH_NUMBER" ]; then
        HIGHEST=$(get_highest_from_specs "$SPECS_DIR")
        BRANCH_NUMBER=$((HIGHEST + 1))
    fi

    # Force base-10 interpretation to prevent octal conversion (e.g., 010 → 8 in octal, but should be 10 in decimal)
    FEATURE_NUM=$(printf "%03d" "$((10#$BRANCH_NUMBER))")
    BRANCH_NAME="${FEATURE_NUM}-${BRANCH_SUFFIX}"
fi

# GitHub enforces a 244-byte limit on branch names
# Validate and truncate if necessary
MAX_BRANCH_LENGTH=244
if [ ${#BRANCH_NAME} -gt $MAX_BRANCH_LENGTH ]; then
    # Calculate how much we need to trim from suffix
    # Account for prefix length: timestamp (15) + hyphen (1) = 16, or sequential (3) + hyphen (1) = 4
    PREFIX_LENGTH=$(( ${#FEATURE_NUM} + 1 ))
    MAX_SUFFIX_LENGTH=$((MAX_BRANCH_LENGTH - PREFIX_LENGTH))

    # Truncate suffix at word boundary if possible
    TRUNCATED_SUFFIX=$(echo "$BRANCH_SUFFIX" | cut -c1-$MAX_SUFFIX_LENGTH)
    # Remove trailing hyphen if truncation created one
    TRUNCATED_SUFFIX=$(echo "$TRUNCATED_SUFFIX" | sed 's/-$//')

    ORIGINAL_BRANCH_NAME="$BRANCH_NAME"
    BRANCH_NAME="${FEATURE_NUM}-${TRUNCATED_SUFFIX}"

    >&2 echo "[specify] Warning: Branch name exceeded GitHub's 244-byte limit"
    >&2 echo "[specify] Original: $ORIGINAL_BRANCH_NAME (${#ORIGINAL_BRANCH_NAME} bytes)"
    >&2 echo "[specify] Truncated to: $BRANCH_NAME (${#BRANCH_NAME} bytes)"
fi

FEATURE_DIR="$SPECS_DIR/$BRANCH_NAME"
SPEC_FILE="$FEATURE_DIR/spec.md"

BASE_BRANCH="${BASE_BRANCH:-develop}"
GIT_SUFFIX="${BRANCH_SUFFIX:-feature}"
GIT_BRANCH="${BRANCH_TYPE}/${GIT_SUFFIX}"
if [ ${#GIT_BRANCH} -gt $MAX_BRANCH_LENGTH ]; then
    GIT_PREFIX_LENGTH=$(( ${#BRANCH_TYPE} + 1 ))
    MAX_GIT_SUFFIX_LENGTH=$((MAX_BRANCH_LENGTH - GIT_PREFIX_LENGTH))
    GIT_TRUNCATED_SUFFIX=$(echo "$GIT_SUFFIX" | cut -c1-$MAX_GIT_SUFFIX_LENGTH | sed 's/-$//')
    GIT_BRANCH="${BRANCH_TYPE}/${GIT_TRUNCATED_SUFFIX}"
fi

if [ "$DRY_RUN" != true ]; then
    if [ -d "$FEATURE_DIR" ] && [ "$ALLOW_EXISTING" != true ]; then
        if [ "$USE_TIMESTAMP" = true ]; then
            >&2 echo "Error: Feature directory '$FEATURE_DIR' already exists. Rerun to get a new timestamp or use a different --short-name."
        else
            >&2 echo "Error: Feature directory '$FEATURE_DIR' already exists. Please use a different feature name or specify a different number with --number."
        fi
        exit 1
    fi

    if [ "$CREATE_BRANCH" = true ]; then
        if ! git rev-parse --is-inside-work-tree >/dev/null 2>&1; then
            echo "Error: --create-branch requires running inside a Git repository." >&2
            exit 1
        fi

        CURRENT_GIT_BRANCH=$(git symbolic-ref --quiet --short HEAD 2>/dev/null || true)

        if [ "$CURRENT_GIT_BRANCH" = "master" ] && [ "$EXPLICIT_BASE" != true ]; then
            echo "Error: Currently on 'master'. Refusing to branch from 'master'; switch to 'develop' or pass --base develop (or --base master explicitly)." >&2
            exit 1
        fi
        if [ "$BASE_BRANCH" = "master" ] && [ "$EXPLICIT_BASE" != true ]; then
            echo "Error: Refusing to branch from 'master' unless --base master is explicitly specified." >&2
            exit 1
        fi

        if [ -n "$(git status --porcelain)" ]; then
            echo "Error: Working tree has uncommitted changes. Commit or stash them before creating branch '$GIT_BRANCH'." >&2
            exit 1
        fi

        if ! git rev-parse --verify --quiet "refs/heads/$BASE_BRANCH" >/dev/null; then
            echo "Error: Base branch '$BASE_BRANCH' does not exist locally." >&2
            exit 1
        fi

        if [ "$CURRENT_GIT_BRANCH" = "$GIT_BRANCH" ]; then
            if [ "$ALLOW_EXISTING" != true ]; then
                echo "Error: Already on Git branch '$GIT_BRANCH'. Pass --allow-existing-branch to reuse it." >&2
                exit 1
            fi
            echo "[specify] Reusing current Git branch '$GIT_BRANCH'" >&2
        elif git rev-parse --verify --quiet "refs/heads/$GIT_BRANCH" >/dev/null; then
            if [ "$ALLOW_EXISTING" != true ]; then
                echo "Error: Git branch '$GIT_BRANCH' already exists. Pass --allow-existing-branch to check it out." >&2
                exit 1
            fi
            git checkout "$GIT_BRANCH" >&2
        else
            git checkout -b "$GIT_BRANCH" "$BASE_BRANCH" >&2
        fi
    fi

    mkdir -p "$SPECS_DIR"
    mkdir -p "$FEATURE_DIR"

    if [ ! -f "$SPEC_FILE" ]; then
        TEMPLATE=$(resolve_template "spec-template" "$REPO_ROOT") || true
        if [ -n "$TEMPLATE" ] && [ -f "$TEMPLATE" ]; then
            cp "$TEMPLATE" "$SPEC_FILE"
        else
            echo "Warning: Spec template not found; created empty spec file" >&2
            touch "$SPEC_FILE"
        fi
    fi

    # Persist to .specify/feature.json so downstream commands can find the feature
    _persist_feature_json "$REPO_ROOT" "$FEATURE_DIR"

    # Inform the user how to set feature state in their own shell
    printf '# To persist: export SPECIFY_FEATURE=%q\n' "$BRANCH_NAME" >&2
    printf '#              export SPECIFY_FEATURE_DIRECTORY=%q\n' "$FEATURE_DIR" >&2
fi

if [ "$CREATE_BRANCH" != true ] && git rev-parse --is-inside-work-tree >/dev/null 2>&1; then
    CURRENT_GIT_BRANCH=$(git symbolic-ref --quiet --short HEAD 2>/dev/null || true)
    if [ "$CURRENT_GIT_BRANCH" = "master" ]; then
        printf '[specify] Warning: Currently on '\''master'\''. New features must branch off '\''develop'\'': git checkout -b %q develop\n' "$GIT_BRANCH" >&2
    fi
    if [ "$DRY_RUN" != true ]; then
        printf '# Git Flow branch (from %s): git checkout -b %q %q\n' "$BASE_BRANCH" "$GIT_BRANCH" "$BASE_BRANCH" >&2
    fi
fi

if $JSON_MODE; then
    if command -v jq >/dev/null 2>&1; then
        if [ "$DRY_RUN" = true ]; then
            jq -cn \
                --arg branch_name "$BRANCH_NAME" \
                --arg spec_file "$SPEC_FILE" \
                --arg feature_num "$FEATURE_NUM" \
                --arg git_branch "$GIT_BRANCH" \
                --arg base_branch "$BASE_BRANCH" \
                '{BRANCH_NAME:$branch_name,SPEC_FILE:$spec_file,FEATURE_NUM:$feature_num,GIT_BRANCH:$git_branch,BASE_BRANCH:$base_branch,DRY_RUN:true}'
        else
            jq -cn \
                --arg branch_name "$BRANCH_NAME" \
                --arg spec_file "$SPEC_FILE" \
                --arg feature_num "$FEATURE_NUM" \
                --arg git_branch "$GIT_BRANCH" \
                --arg base_branch "$BASE_BRANCH" \
                '{BRANCH_NAME:$branch_name,SPEC_FILE:$spec_file,FEATURE_NUM:$feature_num,GIT_BRANCH:$git_branch,BASE_BRANCH:$base_branch}'
        fi
    else
        if [ "$DRY_RUN" = true ]; then
            printf '{"BRANCH_NAME":"%s","SPEC_FILE":"%s","FEATURE_NUM":"%s","GIT_BRANCH":"%s","BASE_BRANCH":"%s","DRY_RUN":true}\n' \
                "$(json_escape "$BRANCH_NAME")" "$(json_escape "$SPEC_FILE")" "$(json_escape "$FEATURE_NUM")" "$(json_escape "$GIT_BRANCH")" "$(json_escape "$BASE_BRANCH")"
        else
            printf '{"BRANCH_NAME":"%s","SPEC_FILE":"%s","FEATURE_NUM":"%s","GIT_BRANCH":"%s","BASE_BRANCH":"%s"}\n' \
                "$(json_escape "$BRANCH_NAME")" "$(json_escape "$SPEC_FILE")" "$(json_escape "$FEATURE_NUM")" "$(json_escape "$GIT_BRANCH")" "$(json_escape "$BASE_BRANCH")"
        fi
    fi
else
    echo "BRANCH_NAME: $BRANCH_NAME"
    echo "SPEC_FILE: $SPEC_FILE"
    echo "FEATURE_NUM: $FEATURE_NUM"
    echo "GIT_BRANCH: $GIT_BRANCH"
    echo "BASE_BRANCH: $BASE_BRANCH"
    if [ "$DRY_RUN" != true ]; then
        printf '# To persist in your shell: export SPECIFY_FEATURE=%q\n' "$BRANCH_NAME"
        printf '#                           export SPECIFY_FEATURE_DIRECTORY=%q\n' "$FEATURE_DIR"
    fi
fi
