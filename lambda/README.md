# NES Outage Lambda Functions

## Functions

### 1. Data Archiver (`nes_archiver.py`)
Fetches NES outage data every 10 minutes and saves it to S3.

### 2. Health Check (`health_check.py`)
API endpoint to validate the NES API is accessible and returning expected data.

**Endpoint:** `GET /health`

**Checks performed:**
1. **API Reachability** - Can we connect to the NES API?
2. **JSON Parseability** - Is the response valid JSON?
3. **Status Field Validation** - Do events contain the 'status' field?

**Response (200 OK):**
```json
{
  "healthy": true,
  "checks": {
    "api_reachable": true,
    "json_parseable": true,
    "status_field_present": true
  },
  "errors": null,
  "event_count": 42,
  "sample_status_values": ["Assigned", "Unassigned"]
}
```

**Response (503 Service Unavailable):**
```json
{
  "healthy": false,
  "checks": {
    "api_reachable": true,
    "json_parseable": true,
    "status_field_present": false
  },
  "errors": ["Status field missing in 3/5 sampled events"],
  "event_count": 42
}
```

---

# NES Outage Data Archiver

AWS Lambda function that fetches NES outage data every 10 minutes and saves it to S3.

## Data Structure

Files are saved with the path: `YYYY/MM/DD/HHMM.json`

```json
{
  "timestamp": "2026-01-29T12:30:00.000Z",
  "event_count": 42,
  "events": [...]
}
```

## Quick Deploy (SAM CLI)

```bash
# Install SAM CLI if needed
# brew install aws-sam-cli

# Deploy with default bucket name
sam build && sam deploy --guided

# Or specify bucket name
sam build && sam deploy --parameter-overrides BucketName=my-nes-archive
```

## Manual Setup

### 1. Create S3 Bucket

```bash
aws s3 mb s3://nes-outage-archive
```

### 2. Create Lambda Function

```bash
# Zip the code
zip nes_archiver.zip nes_archiver.py

# Create the function
aws lambda create-function \
  --function-name nes-outage-archiver \
  --runtime python3.11 \
  --handler nes_archiver.lambda_handler \
  --role arn:aws:iam::YOUR_ACCOUNT:role/YOUR_LAMBDA_ROLE \
  --zip-file fileb://nes_archiver.zip \
  --timeout 30 \
  --environment "Variables={BUCKET_NAME=nes-outage-archive}"
```

### 3. Create IAM Role

The Lambda needs a role with:
- `AWSLambdaBasicExecutionRole` (for CloudWatch logs)
- S3 write access to your bucket:

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": ["s3:PutObject"],
      "Resource": "arn:aws:s3:::nes-outage-archive/*"
    }
  ]
}
```

### 4. Create EventBridge Schedule

```bash
# Create rule
aws events put-rule \
  --name nes-archiver-schedule \
  --schedule-expression "rate(10 minutes)"

# Add Lambda as target
aws events put-targets \
  --rule nes-archiver-schedule \
  --targets "Id"="1","Arn"="arn:aws:lambda:REGION:ACCOUNT:function:nes-outage-archiver"

# Grant EventBridge permission to invoke Lambda
aws lambda add-permission \
  --function-name nes-outage-archiver \
  --statement-id eventbridge-invoke \
  --action lambda:InvokeFunction \
  --principal events.amazonaws.com \
  --source-arn arn:aws:events:REGION:ACCOUNT:rule/nes-archiver-schedule
```

## Query Archived Data

```bash
# List files for a specific day
aws s3 ls s3://nes-outage-archive/2026/01/29/

# Download a specific snapshot
aws s3 cp s3://nes-outage-archive/2026/01/29/1230.json ./

# Sync all data locally
aws s3 sync s3://nes-outage-archive ./archive/
```

## Analyze with Athena (Optional)

You can query the archived JSON data with Athena:

```sql
CREATE EXTERNAL TABLE nes_outages (
  timestamp string,
  event_count int,
  events array<struct<
    id: int,
    status: string,
    numPeople: int,
    cause: string,
    title: string,
    startTime: bigint,
    lastUpdatedTime: bigint,
    latitude: double,
    longitude: double
  >>
)
ROW FORMAT SERDE 'org.openx.data.jsonserde.JsonSerDe'
LOCATION 's3://nes-outage-archive/';

-- Example: Find events with most affected people
SELECT
  event.id,
  event.status,
  event.numPeople,
  event.cause,
  timestamp
FROM nes_outages
CROSS JOIN UNNEST(events) AS t(event)
ORDER BY event.numPeople DESC
LIMIT 20;
```

## Cost Estimate

- Lambda: ~4,320 invocations/month = ~$0.01
- S3: ~2MB/day × 90 days = ~180MB = ~$0.01/month
- EventBridge: Free tier covers this

**Total: < $0.05/month**
