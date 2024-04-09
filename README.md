# Google Indexing API

This project uses the Google Indexing API to notify Google of new or updated URLs on your website.

## Setup

Environment variables are used to store the configuration for the project. The following environment variables are required:

```
GOOGLE_APPLICATION_CREDENTIALS - The path to the service account key file
SITEMAP_FILE - The path to the sitemap file
INDEXED_FILE - The path to the CSV file that stores the already indexed URLs (you can get it from the Google Search Console)
SENT_FILE - The path to the CSV file that stores the already sent URLs. It will be created if it doesn't exist
RATE_LIMIT_PER_DAY - The number of requests allowed per day, Default: 200
RATE_LIMIT_PER_MINUTE - The number of requests allowed per minute, Default: 60

```
