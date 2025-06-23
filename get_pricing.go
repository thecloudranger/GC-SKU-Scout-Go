/*
Copyright 2023 Nils Knieling. All Rights Reserved.
Copyright 2023 Roman Inflianskas. All Rights Reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	billing "cloud.google.com/go/billing/apiv1"
	"cloud.google.com/go/billing/apiv1/billingpb"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	"gopkg.in/yaml.v3"
)

var (
	region = flag.String("region", "me-central2", "Google Cloud region")
)

type Sku struct {
	Name                     string
	SkuId                    string
	Description              string
	ServiceDisplayName       string
	ResourceFamily           string
	ResourceGroup            string
	UsageType                string
	ServiceRegions           []string
	PricingInfo              []*billingpb.PricingInfo
	ServiceProviderName      string
	GeoTaxonomy              *billingpb.GeoTaxonomy
	Mapping                  string
	Nanos                    int32
	Units                    int64
	CurrencyCode             string
	UsageUnit                string
	UsageUnitDescription     string
	BaseUnit                 string
	BaseUnitDescription      string
	BaseUnitConversionFactor float64
	DisplayQuantity          float64
	CalculatedPrice          float64
	PricePerUnit             string
}

type GcpConfig struct {
	Region map[string]interface{} `yaml:"region"`
}

func main() {
	flag.Parse()

	apiKey := os.Getenv("API_KEY")
	if apiKey == "" {
		log.Fatalf("ERROR: API_KEY environment variable not set.")
	}

	gcpFile, err := os.ReadFile("gcp.yml")
	if err != nil {
		log.Fatalf("ERROR: Cannot read gcp.yml: %v", err)
	}
	var gcpConfig GcpConfig
	if err := yaml.Unmarshal(gcpFile, &gcpConfig); err != nil {
		log.Fatalf("ERROR: Cannot unmarshal gcp.yml: %v", err)
	}

	if _, ok := gcpConfig.Region[*region]; !ok {
		log.Fatalf("ERROR: Region '%s' not found in gcp.yml", *region)
	}

	fmt.Printf("Fetching pricing for region: %s\n", *region)

	ctx := context.Background()
	c, err := billing.NewCloudCatalogClient(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		log.Fatalf("ERROR: Cannot create Google Cloud Billing client: %v", err)
	}
	defer c.Close()

	serviceIds := []string{
		"6F81-5844-456A", // Compute Engine
		"E505-1604-58F8", // Networking
		"95FF-2EF5-5EA1", // Cloud Storage
		"58CD-E7C3-72CA", // Cloud Monitoring
		"9662-B51E-5089", // Cloud SQL
		"CCD8-9BF1-090E", //Kubernetes Engine
		"5490-F7B7-8DF6", //Cloud Logging

	}

	var allSkus []Sku

	for _, serviceId := range serviceIds {
		fmt.Printf("Fetching SKUs for service: %s\n", serviceId)
		req := &billingpb.ListSkusRequest{
			Parent: fmt.Sprintf("services/%s", serviceId),
		}
		it := c.ListSkus(ctx, req)
		for {
			sku, err := it.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				// This is not a fatal error, so we log it and continue.
				log.Printf("WARN: Error fetching SKU, skipping: %v", err)
				continue
			}

			isRegionFound := false
			for _, r := range sku.ServiceRegions {
				if r == *region || r == "global" || r == "multi-region" {
					isRegionFound = true
					break
				}
			}

			if isRegionFound {
				var nanos int32
				var units int64
				var currencyCode string
				var calculatedPrice float64
				var pricePerUnit string
				if len(sku.PricingInfo) > 0 && len(sku.PricingInfo[0].PricingExpression.TieredRates) > 0 {
					nanos = sku.PricingInfo[0].PricingExpression.TieredRates[0].UnitPrice.Nanos
					units = sku.PricingInfo[0].PricingExpression.TieredRates[0].UnitPrice.Units
					currencyCode = sku.PricingInfo[0].PricingExpression.TieredRates[0].UnitPrice.CurrencyCode
					calculatedPrice = float64(units) + float64(nanos)/1e9
					pricePerUnit = fmt.Sprintf("%.10f %s per %s", calculatedPrice, currencyCode, sku.PricingInfo[0].PricingExpression.UsageUnitDescription)
				}

				allSkus = append(allSkus, Sku{
					Name:                     sku.Name,
					SkuId:                    sku.SkuId,
					Description:              sku.Description,
					ServiceDisplayName:       sku.Category.ServiceDisplayName,
					ResourceFamily:           sku.Category.ResourceFamily,
					ResourceGroup:            sku.Category.ResourceGroup,
					UsageType:                sku.Category.UsageType,
					ServiceRegions:           sku.ServiceRegions,
					PricingInfo:              sku.PricingInfo,
					ServiceProviderName:      sku.ServiceProviderName,
					GeoTaxonomy:              sku.GeoTaxonomy,
					Nanos:                    nanos,
					Units:                    units,
					CurrencyCode:             currencyCode,
					CalculatedPrice:          calculatedPrice,
					PricePerUnit:             pricePerUnit,
					UsageUnit:                sku.PricingInfo[0].PricingExpression.UsageUnit,
					UsageUnitDescription:     sku.PricingInfo[0].PricingExpression.UsageUnitDescription,
					BaseUnit:                 sku.PricingInfo[0].PricingExpression.BaseUnit,
					BaseUnitDescription:      sku.PricingInfo[0].PricingExpression.BaseUnitDescription,
					BaseUnitConversionFactor: sku.PricingInfo[0].PricingExpression.BaseUnitConversionFactor,
					DisplayQuantity:          sku.PricingInfo[0].PricingExpression.DisplayQuantity,
				})
			}
		}
		time.Sleep(3 * time.Second) // To avoid hitting API rate limits
	}

	// Marshal to JSON
	jsonData, err := json.MarshalIndent(allSkus, "", "  ")
	if err != nil {
		log.Fatalf("ERROR: Cannot marshal to JSON: %v", err)
	}

	// Create filename with date and time
	filename := fmt.Sprintf("pricing-%s-%s.json", *region, time.Now().Format("2006-01-02-15-04-05"))

	// Write to file
	err = os.WriteFile(filename, jsonData, 0644)
	if err != nil {
		log.Fatalf("ERROR: Cannot write to file: %v", err)
	}

	fmt.Printf("\nPricing information saved to %s\n", filename)
	fmt.Printf("\nFound %d SKUs for region %s\n", len(allSkus), *region)
}
