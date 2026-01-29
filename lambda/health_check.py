"""
NES Outage API Health Check
Lambda function to validate NES API is accessible and returning expected data.

Returns:
- 200: All checks passed
- 503: One or more checks failed

Checks:
1. API Reachability - Can we connect to the API?
2. JSON Parseability - Is the response valid JSON?
3. Status Field Validation - Do events contain the 'status' field?
"""

import json
import urllib.request
import urllib.error

API_URL = "https://utilisocial.io/datacapable/v2/p/NES/map/events"


def lambda_handler(event, context):
    """Health check endpoint for NES Outage API."""

    checks = {
        "api_reachable": False,
        "json_parseable": False,
        "status_field_present": False
    }
    errors = []

    # Check 1: API Reachability
    try:
        req = urllib.request.Request(
            API_URL,
            headers={"User-Agent": "NES-HealthCheck/1.0"}
        )
        with urllib.request.urlopen(req, timeout=10) as response:
            if response.status == 200:
                checks["api_reachable"] = True
                raw_data = response.read().decode("utf-8")
            else:
                errors.append(f"API returned status {response.status}")
                raw_data = None
    except urllib.error.URLError as e:
        errors.append(f"API unreachable: {str(e)}")
        raw_data = None
    except Exception as e:
        errors.append(f"Connection error: {str(e)}")
        raw_data = None

    # Check 2: JSON Parseability
    data = None
    if raw_data:
        try:
            data = json.loads(raw_data)
            if isinstance(data, list):
                checks["json_parseable"] = True
            else:
                errors.append("API response is not a JSON array")
        except json.JSONDecodeError as e:
            errors.append(f"Invalid JSON: {str(e)}")

    # Check 3: Status Field Validation
    if data and len(data) > 0:
        # Check first few events for status field
        events_with_status = 0
        sample_size = min(5, len(data))

        for event in data[:sample_size]:
            if "status" in event:
                events_with_status += 1

        if events_with_status == sample_size:
            checks["status_field_present"] = True
        else:
            errors.append(f"Status field missing in {sample_size - events_with_status}/{sample_size} sampled events")
    elif data is not None and len(data) == 0:
        # Empty array is valid (no current outages)
        checks["status_field_present"] = True

    # Determine overall health
    all_passed = all(checks.values())
    status_code = 200 if all_passed else 503

    response_body = {
        "healthy": all_passed,
        "checks": checks,
        "errors": errors if errors else None,
        "event_count": len(data) if data else 0,
        "api_url": API_URL
    }

    # Add sample status values if available
    if data and len(data) > 0:
        status_values = list(set(e.get("status", "N/A") for e in data[:10]))
        response_body["sample_status_values"] = status_values

    return {
        "statusCode": status_code,
        "headers": {
            "Content-Type": "application/json",
            "Access-Control-Allow-Origin": "*"
        },
        "body": json.dumps(response_body, indent=2)
    }


# For local testing
if __name__ == "__main__":
    result = lambda_handler({}, None)
    print(f"Status: {result['statusCode']}")
    print(result['body'])
