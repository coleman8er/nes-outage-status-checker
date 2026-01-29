#!/bin/bash
# Count events in each archived JSON file from S3
#
# Usage: ./count_events.sh [date]
# Example: ./count_events.sh 2026/01/29
#          ./count_events.sh  (defaults to today)

BUCKET="nes-outage-status-checker-archive"
PROFILE="personal"
DATE="${1:-$(date +%Y/%m/%d)}"

echo "Checking s3://$BUCKET/$DATE/"
echo "=========================================="

total_files=0
total_events=0

for file in $(aws s3 ls "s3://$BUCKET/$DATE/" --profile "$PROFILE" | awk '{print $4}'); do
    # Download and parse JSON
    content=$(aws s3 cp "s3://$BUCKET/$DATE/$file" - --profile "$PROFILE" 2>/dev/null)

    if [ -n "$content" ]; then
        event_count=$(echo "$content" | jq '.event_count // (.events | length) // 0')
        timestamp=$(echo "$content" | jq -r '.timestamp // "N/A"')

        printf "%-12s | Events: %4s | %s\n" "$file" "$event_count" "$timestamp"

        total_files=$((total_files + 1))
        total_events=$((total_events + event_count))
    fi
done

echo "=========================================="
echo "Total files: $total_files"
echo "Total events across all snapshots: $total_events"
if [ $total_files -gt 0 ]; then
    echo "Average events per snapshot: $((total_events / total_files))"
fi
