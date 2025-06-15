---
title: "API Endpoints"
author: "LogSum Team"
date: "2024-01-15"
tags: ["api", "reference", "endpoints"]
language: "en"
---

# API Endpoints

LogSum provides a comprehensive REST API for programmatic access to log analysis functionality.

## Authentication

All API requests require authentication using API keys:

```bash
curl -H "Authorization: Bearer YOUR_API_KEY" https://api.logsum.io/v1/analyze
```

## Core Endpoints

### POST /v1/analyze

Analyzes log data and returns insights.

**Request:**
```json
{
  "logs": ["2024-01-01 10:00:00 ERROR Database connection failed"],
  "options": {
    "detect_patterns": true,
    "find_anomalies": true,
    "generate_insights": true
  }
}
```

**Response:**
```json
{
  "patterns": [
    {
      "id": "pattern_1",
      "template": "Database connection failed",
      "count": 15,
      "severity": "error"
    }
  ],
  "anomalies": [],
  "insights": [
    {
      "type": "performance",
      "message": "High error rate detected in database connections"
    }
  ]
}
```

### GET /v1/patterns

Retrieves stored log patterns.

**Parameters:**
- `limit` (optional): Maximum number of patterns to return
- `severity` (optional): Filter by severity level

**Response:**
```json
{
  "patterns": [
    {
      "id": "pattern_1",
      "template": "Database connection failed",
      "count": 15,
      "last_seen": "2024-01-15T10:00:00Z"
    }
  ],
  "total": 1
}
```

### POST /v1/watch

Starts real-time log monitoring.

**Request:**
```json
{
  "source": "/var/log/app.log",
  "filters": {
    "severity": ["error", "warning"]
  },
  "webhook_url": "https://your-app.com/webhook"
}
```

## Error Handling

All endpoints return standard HTTP status codes:

- `200 OK` - Success
- `400 Bad Request` - Invalid request format
- `401 Unauthorized` - Missing or invalid API key
- `429 Too Many Requests` - Rate limit exceeded
- `500 Internal Server Error` - Server error

Error responses include details:

```json
{
  "error": {
    "code": "invalid_request",
    "message": "Missing required field: logs"
  }
}
```

## Rate Limiting

API requests are limited to:
- 100 requests per minute for free tier
- 1000 requests per minute for pro tier
- 10000 requests per minute for enterprise tier

Rate limit headers are included in responses:
- `X-RateLimit-Limit`: Request limit per window
- `X-RateLimit-Remaining`: Requests remaining in current window
- `X-RateLimit-Reset`: Timestamp when window resets