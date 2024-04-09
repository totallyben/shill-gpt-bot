#!/bin/bash

# The Twitter URL to fetch the content from
URL="$1"

# Use curl to fetch the HTML content of the page
HTML_CONTENT=$(curl -sL "$URL")

# Attempt to parse the title or content from the HTML
# This is a very basic and fragile approach and may not work consistently
TITLE=$(echo "$HTML_CONTENT" | grep -oP '(?<=<title>)(.*)(?=</title>)' | sed 's/^[ \t]*//;s/[ \t]*$//')

# Echo out the fetched content
echo "Title: $TITLE"

# You might need to use more sophisticated parsing to extract the actual tweet content,
# as it's likely loaded dynamically with JavaScript and not present in the initial HTML response.
