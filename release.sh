#/bin/bash

if [ -z "$1" ]
  then
    echo "$0 patch|minor|major"
    exit 1
fi

BRANCH=$(git rev-parse --abbrev-ref HEAD)
if [[ "$BRANCH" != "master" ]]; then
  echo 'Not on master branch, aborting script';
  exit 1;
fi

# only works for ssh url
OWNER=$(git remote get-url origin | cut -d: -f 2 | cut -d/ -f 1)
REPO=$(git remote get-url origin | cut -d: -f 2 | cut -d/ -f 2)

read -p "Repository [$OWNER/$REPO]: " TMP
if [ ! -z "$TMP" ]; then
  OWNER=$(echo $TMP | cut -d/ -f 1)
  REPO=$(echo $TMP | cut -sd/ -f 2)
fi

if [ -z "$OWNER" ] || [ -z "$REPO" ]; then
  echo "Expected github owner/repo, got '$OWNER/$REPO' - Aborting."
  exit 1
fi

read -p "Your Github username: " USERNAME
read -p "Your Github Token: " TOKEN
echo -en "\033[1A\033[2K"
echo "Your Github Token: ******************"

URL="https://api.github.com/repos/$OWNER/$REPO/collaborators/$USERNAME/permission"
HTTP_RESPONSE=$(curl -s -w "HTTPSTATUS:%{http_code}" -X GET -u $USERNAME:$TOKEN $URL)
HTTP_BODY=$(echo $HTTP_RESPONSE | sed -e 's/HTTPSTATUS\:.*//g')
HTTP_STATUS=$(echo $HTTP_RESPONSE | tr -d '\n' | sed -e 's/.*HTTPSTATUS://')

PERMISSION=""
if [ $HTTP_STATUS -eq 200  ]; then
  PERMISSION=$(echo $HTTP_BODY | python2 -c 'import json,sys;res=json.load(sys.stdin); print res["permission"]')
fi

if [ $PERMISSION != "admin" ] && [ $PERMISSION != "write" ]; then
  echo "Missing write/admin permission to repo, Aborting."
  exit 1
fi

URL="https://api.github.com/repos/$OWNER/$REPO/releases/latest"
HTTP_RESPONSE=$(curl -s -w "HTTPSTATUS:%{http_code}" -X GET -u $USERNAME:$TOKEN $URL)
HTTP_BODY=$(echo $HTTP_RESPONSE | sed -e 's/HTTPSTATUS\:.*//g')
HTTP_STATUS=$(echo $HTTP_RESPONSE | tr -d '\n' | sed -e 's/.*HTTPSTATUS://')

if [ $HTTP_STATUS -eq 200  ]; then
  CURRENT_RELEASE=$(echo $HTTP_BODY | python2 -c 'import json,sys;res=json.load(sys.stdin); print res["tag_name"]')

  TARGET_COMITISH=$(echo $HTTP_BODY | python2 -c 'import json,sys;res=json.load(sys.stdin); print res["target_commitish"]')
fi

NEW_RELEASE="0.0.0"
if [ $1 == "major"  ]; then
  NEW_RELEASE=$(echo "${CURRENT_RELEASE:-0.0.-1}" | awk -F'[.]' '{print $1+1"."0"."0}')
elif [ $1 == "minor" ]; then
  NEW_RELEASE=$(echo "${CURRENT_RELEASE:-0.0.-1}" | awk -F'[.]' '{print $1"."$2+1"."0}')
elif [ $1 == "patch" ]; then
  NEW_RELEASE=$(echo "${CURRENT_RELEASE:-0.0.-1}" | awk -F'[.]' '{print $1"."$2"."$3+1}')
fi

LAST_COMMIT=$(git rev-parse HEAD)
if [ ! -z $TARGET_COMITISH ] && [ $TARGET_COMITISH == $LAST_COMMIT ]; then
  echo "Latest commit in master is the same as the latest release, '$TARGET_COMITISH', Aborting."
  exit 1
fi

echo "Current release : $CURRENT_RELEASE"
echo "New release     : $NEW_RELEASE"

read -p "Continue? [y/n]: " CONTINUE
if [ $CONTINUE != "y" ]; then
  echo "Aborting."
  exit 1
fi

URL="https://api.github.com/repos/$OWNER/$REPO/releases"
DATA="{\"tag_name\":\"$NEW_RELEASE\",\"target_commitish\":\"$LAST_COMMIT\"}"
HTTP_RESPONSE=$(curl -s -d "$DATA" -w "HTTPSTATUS:%{http_code}" -X POST -u $USERNAME:$TOKEN $URL)
HTTP_BODY=$(echo $HTTP_RESPONSE | sed -e 's/HTTPSTATUS\:.*//g')
HTTP_STATUS=$(echo $HTTP_RESPONSE | tr -d '\n' | sed -e 's/.*HTTPSTATUS://')

echo $HTTP_BODY