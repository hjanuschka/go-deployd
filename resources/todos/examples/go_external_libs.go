// Example: Go Event Script with External Libraries
// This demonstrates various external Go libraries that can be used in event scripts

import (
    "strings"
    "time"
    "github.com/google/uuid"
    "github.com/shopspring/decimal"
    "golang.org/x/crypto/bcrypt"
)

func Run(ctx *EventContext) error {
    // 1. UUID Generation (github.com/google/uuid)
    if ctx.Data["generateId"] == true {
        uniqueID := uuid.New().String()
        ctx.Data["uuid"] = uniqueID
        ctx.Data["shortId"] = uniqueID[:8]
    }

    // 2. Precise Decimal Calculations (github.com/shopspring/decimal)
    if price, ok := ctx.Data["price"].(float64); ok {
        priceDecimal := decimal.NewFromFloat(price)
        
        // Calculate tax (7.5%)
        taxRate := decimal.NewFromFloat(0.075)
        tax := priceDecimal.Mul(taxRate)
        total := priceDecimal.Add(tax)
        
        ctx.Data["tax"] = tax.InexactFloat64()
        ctx.Data["totalWithTax"] = total.InexactFloat64()
        ctx.Data["taxRate"] = "7.5%"
    }

    // 3. Password Hashing (golang.org/x/crypto/bcrypt)
    if password, ok := ctx.Data["password"].(string); ok && password != "" {
        hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
        if err != nil {
            ctx.Error("password", "Failed to hash password")
            return nil
        }
        ctx.Data["hashedPassword"] = string(hashedPassword)
        // Remove plain password from data
        delete(ctx.Data, "password")
    }

    // 4. String Processing with external utilities
    if title, ok := ctx.Data["title"].(string); ok {
        // Clean and format title
        cleanTitle := strings.TrimSpace(title)
        ctx.Data["titleLength"] = len(cleanTitle)
        ctx.Data["titleWords"] = len(strings.Fields(cleanTitle))
        ctx.Data["titleSlug"] = strings.ToLower(strings.ReplaceAll(cleanTitle, " ", "-"))
    }

    // 5. Timestamp handling
    now := time.Now()
    ctx.Data["createdAt"] = now.Format(time.RFC3339)
    ctx.Data["createdAtUnix"] = now.Unix()
    ctx.Data["createdAtUTC"] = now.UTC().Format("2006-01-02 15:04:05")

    // 6. Conditional logic based on user context
    if ctx.Me != nil {
        ctx.Data["createdBy"] = ctx.Me["id"]
        ctx.Data["createdByName"] = ctx.Me["name"]
    } else {
        ctx.Cancel("Authentication required", 401)
    }

    // 7. Data validation with external helpers
    if email, ok := ctx.Data["email"].(string); ok {
        if !strings.Contains(email, "@") || !strings.Contains(email, ".") {
            ctx.Error("email", "Invalid email format")
        }
    }

    return nil
}

/*
To use this script:

1. Add to resources/todos/config.json:
{
  "eventConfig": {
    "post": {
      "runtime": "go"
    }
  }
}

2. The server will automatically download and compile the external dependencies:
   - github.com/google/uuid
   - github.com/shopspring/decimal  
   - golang.org/x/crypto

3. Test with:
curl -X POST http://localhost:2405/todos \
  -H "Content-Type: application/json" \
  -d '{
    "title": "Test External Libraries",
    "price": 99.99,
    "password": "mypassword123",
    "email": "user@example.com",
    "generateId": true
  }'

Expected response includes:
- uuid, shortId (from UUID library)
- tax, totalWithTax (from decimal library) 
- hashedPassword (from bcrypt)
- titleSlug, titleWords (from string processing)
- Various timestamps
*/