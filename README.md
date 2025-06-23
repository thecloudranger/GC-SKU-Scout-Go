# SKU-Scout (Go Version)

This program fetches SKU pricing information from the Google Cloud Billing API for a specific region and saves it to a JSON file.

## Prerequisites

*   [Go](https://go.dev/doc/install) installed on your system.
*   A Google Cloud Platform project with the [Cloud Billing API](https://console.cloud.google.com/flows/enableapi?apiid=cloudbilling.googleapis.com) enabled.
*   An API key with access to the Cloud Billing API.

## Setup

1.  **Clone the repository:**
    ```bash
    git clone <your-repo-url>
    cd <your-repo-name>/build
    ```

2.  **Set your API Key:**
    Export your API key as an environment variable.
    ```bash
    export API_KEY="YOUR-CLOUD-BILLING-API-KEY"
    ```

## Running the Program

You can run the program using the `go run` command. Use the `-region` flag to specify the Google Cloud region you want to fetch pricing for.

```bash
go run get_pricing.go -region=me-central2
```

The script will create a JSON file in the `build` directory named `pricing-<region>-<timestamp>.json` with the SKU information.
