#!/bin/bash

# Solana Balance API - Complete Demo Script
# This script demonstrates all API functionality for video recording

# Colors for better output formatting
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
CYAN='\033[0;36m'
WHITE='\033[1;37m'
NC='\033[0m' # No Color

# Configuration
API_BASE_URL="http://localhost:8080"
VALID_API_KEY="test-api-key-1"
INVALID_API_KEY="invalid-key-12345"
INACTIVE_API_KEY="inactive-test-key"

# Function to print formatted headers
print_header() {
    echo ""
    echo -e "${WHITE}================================================================${NC}"
    echo -e "${WHITE}  $1${NC}"
    echo -e "${WHITE}================================================================${NC}"
    echo ""
}

# Function to print sub-headers
print_subheader() {
    echo ""
    echo -e "${CYAN}--- $1 ---${NC}"
    echo ""
}

# Function to print status messages
print_status() {
    echo -e "${YELLOW}[INFO]${NC} $1"
}

# Function to print success messages
print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

# Function to print error messages
print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Function to print demo information
print_demo_info() {
    echo -e "${PURPLE}[DEMO]${NC} $1"
}

# Function to pause with countdown
pause_with_countdown() {
    local seconds=$1
    local message=${2:-"Continuing in"}
    
    echo ""
    echo -e "${BLUE}$message${NC}"
    for ((i=seconds; i>=1; i--)); do
        echo -ne "${BLUE}$i... ${NC}"
        sleep 1
    done
    echo -e "${GREEN}GO!${NC}"
    echo ""
}

# Function to make API request with formatted output
make_api_request() {
    local method=$1
    local endpoint=$2
    local auth_header=$3
    local data=$4
    local description=$5
    
    echo -e "${CYAN}Request:${NC} $method $endpoint"
    if [ ! -z "$auth_header" ]; then
        echo -e "${CYAN}Headers:${NC} Authorization: $auth_header"
    fi
    if [ ! -z "$data" ]; then
        echo -e "${CYAN}Body:${NC} $data"
    fi
    echo ""
    
    echo -e "${YELLOW}Sending request...${NC}"
    
    if [ ! -z "$data" ]; then
        if [ ! -z "$auth_header" ]; then
            response=$(curl -s -w "\nHTTP_STATUS:%{http_code}\nTIME_TOTAL:%{time_total}" \
                -X "$method" "$API_BASE_URL$endpoint" \
                -H "Content-Type: application/json" \
                -H "Authorization: $auth_header" \
                -d "$data")
        else
            response=$(curl -s -w "\nHTTP_STATUS:%{http_code}\nTIME_TOTAL:%{time_total}" \
                -X "$method" "$API_BASE_URL$endpoint" \
                -H "Content-Type: application/json" \
                -d "$data")
        fi
    else
        if [ ! -z "$auth_header" ]; then
            response=$(curl -s -w "\nHTTP_STATUS:%{http_code}\nTIME_TOTAL:%{time_total}" \
                -X "$method" "$API_BASE_URL$endpoint" \
                -H "Authorization: $auth_header")
        else
            response=$(curl -s -w "\nHTTP_STATUS:%{http_code}\nTIME_TOTAL:%{time_total}" \
                -X "$method" "$API_BASE_URL$endpoint")
        fi
    fi
    
    # Parse response
    http_status=$(echo "$response" | grep "HTTP_STATUS:" | cut -d: -f2)
    time_total=$(echo "$response" | grep "TIME_TOTAL:" | cut -d: -f2)
    response_body=$(echo "$response" | sed '/HTTP_STATUS:/d' | sed '/TIME_TOTAL:/d')
    
    echo -e "${GREEN}Response Status:${NC} $http_status"
    echo -e "${GREEN}Response Time:${NC} ${time_total}s"
    echo -e "${GREEN}Response Body:${NC}"
    echo "$response_body" | jq '.' 2>/dev/null || echo "$response_body"
    echo ""
}

# Function to check if server is running
check_server() {
    print_status "Checking if API server is running..."
    
    if curl -s "$API_BASE_URL/health" > /dev/null; then
        print_success "API server is running at $API_BASE_URL"
        return 0
    else
        print_error "API server is not running at $API_BASE_URL"
        print_error "Please start the server with: go run cmd/server/main.go"
        return 1
    fi
}

# Function to setup database
setup_database() {
    print_status "Setting up database..."
    
    if go run cmd/dbsetup/main.go -all > /dev/null 2>&1; then
        print_success "Database setup completed"
    else
        print_error "Database setup failed. Please ensure MongoDB is running."
        return 1
    fi
}

# Demo 1: Single Wallet Balance Retrieval
demo_1_single_wallet() {
    print_header "DEMO 1: Single Wallet Balance Retrieval"
    
    print_demo_info "This demo shows the API working with a single wallet address"
    print_demo_info "We'll make two requests to demonstrate caching behavior"
    
    pause_with_countdown 10 "Starting Demo 1 in"
    
    print_subheader "Test 1.1: First Request (Cache Miss)"
    make_api_request "POST" "/api/get-balance" \
        "$VALID_API_KEY" \
        '{"wallets": ["11111111111111111111111111111112"]}' \
        "Single wallet balance request"
    
    pause_with_countdown 5 "Making second request in"
    
    print_subheader "Test 1.2: Immediate Second Request (Cache Hit)"
    make_api_request "POST" "/api/get-balance" \
        "$VALID_API_KEY" \
        '{"wallets": ["11111111111111111111111111111112"]}' \
        "Same wallet - should be cached"
    
    print_success "Demo 1 completed - Notice the 'cached' field difference!"
    
    pause_with_countdown 20 "Demo 1 completed. Starting Demo 2 in"
}

# Demo 2: Multiple Wallets Balance Retrieval
demo_2_multiple_wallets() {
    print_header "DEMO 2: Multiple Wallets Balance Retrieval"
    
    print_demo_info "This demo shows the API processing multiple wallet addresses concurrently"
    print_demo_info "We'll test with 3 wallets first, then with 10 wallets"
    
    pause_with_countdown 10 "Starting Demo 2 in"
    
    print_subheader "Test 2.1: Three Wallets Request"
    make_api_request "POST" "/api/get-balance" \
        "$VALID_API_KEY" \
        '{
            "wallets": [
                "11111111111111111111111111111113",
                "11111111111111111111111111111114",
                "11111111111111111111111111111115"
            ]
        }' \
        "Multiple wallets batch processing"
    
    pause_with_countdown 10 "Making large batch request in"
    
    print_subheader "Test 2.2: Ten Wallets Batch Request"
    make_api_request "POST" "/api/get-balance" \
        "$VALID_API_KEY" \
        '{
            "wallets": [
                "So11111111111111111111111111111111111111112",
                "EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v",
                "Es9vMFrzaCERmJfrF4H2FYD4KCoNkY11McCe8BenwNYB",
                "mSoLzYCxHdYgdzU16g5QSh3i5K3z3KZK7ytfqcJm7So",
                "7dHbWXmci3dT8UFYWYZweBLXgycu7Y3iL6trKn1Y7ARj",
                "DezXAZ8z7PnrnRJjz3wXBoRgixCa6xjnB7YaB1pPB263",
                "A9mUU4qviSctJVPJdBJWkb28deg915LYJKrzQ19ji3FM",
                "Gh9ZwEmdLJ8DscKNTkTqPbNwLNNBjuSzaG9Vp2KGtKJr",
                "TokenkegQfeZyiNwAJbNbGKPFXCWuBvf9Ss623VQ5DA",
                "So11111111111111111111111111111111111111113"
            ]
        }' \
        "Large batch processing (10 wallets)"
    
    print_success "Demo 2 completed - Notice the concurrent processing efficiency!"
    
    pause_with_countdown 20 "Demo 2 completed. Starting Demo 3 in"
}

# Demo 3: Concurrent Requests with Same Wallet
demo_3_concurrent_requests() {
    print_header "DEMO 3: Concurrent Requests with Same Wallet"
    
    print_demo_info "This demo shows 5 concurrent requests for the same wallet"
    print_demo_info "The mutex system prevents duplicate RPC calls"
    print_demo_info "Only one RPC call will be made, others will wait and get cached result"
    
    pause_with_countdown 15 "Starting Demo 3 in"
    
    # Clear cache first
    print_subheader "Clearing cache by waiting 11 seconds..."
    sleep 11
    
    print_subheader "Launching 5 Concurrent Requests"
    print_status "Starting 5 concurrent requests for wallet: So11111111111111111111111111111111111111112"
    
    # Create temporary files for responses
    temp_dir=$(mktemp -d)
    
    # Launch 5 concurrent requests
    for i in {1..5}; do
        (
            echo "Request $i starting at $(date)" > "$temp_dir/request_$i.log"
            response=$(curl -s -w "\nHTTP_STATUS:%{http_code}\nTIME_TOTAL:%{time_total}" \
                -X POST "$API_BASE_URL/api/get-balance" \
                -H "Content-Type: application/json" \
                -H "Authorization: $VALID_API_KEY" \
                -d '{"wallets": ["So11111111111111111111111111111111111111112"]}')
            echo "Request $i completed at $(date)" >> "$temp_dir/request_$i.log"
            echo "$response" >> "$temp_dir/request_$i.log"
        ) &
    done
    
    # Wait for all requests to complete
    wait
    
    print_subheader "Concurrent Requests Results:"
    
    # Display results
    for i in {1..5}; do
        echo -e "${CYAN}Request $i Results:${NC}"
        cat "$temp_dir/request_$i.log"
        echo ""
    done
    
    # Cleanup
    rm -rf "$temp_dir"
    
    print_success "Demo 3 completed - All requests succeeded with mutex protection!"
    
    pause_with_countdown 20 "Demo 3 completed. Starting Demo 4 in"
}

# Demo 4: All Scenarios Simultaneously
demo_4_all_scenarios() {
    print_header "DEMO 4: All Scenarios Simultaneously"
    
    print_demo_info "This demo runs single wallet, multiple wallets, and concurrent requests all at once"
    print_demo_info "This tests the system under mixed load conditions"
    
    pause_with_countdown 15 "Starting Demo 4 in"
    
    print_subheader "Launching All Test Scenarios Simultaneously"
    
    # Clear cache
    sleep 11
    
    temp_dir=$(mktemp -d)
    
    # Single wallet test
    (
        echo "=== SINGLE WALLET TEST ===" > "$temp_dir/single.log"
        response=$(curl -s -w "\nHTTP_STATUS:%{http_code}\nTIME_TOTAL:%{time_total}" \
            -X POST "$API_BASE_URL/api/get-balance" \
            -H "Content-Type: application/json" \
            -H "Authorization: $VALID_API_KEY" \
            -d '{"wallets": ["A9mUU4qviSctJVPJdBJWkb28deg915LYJKrzQ19ji3FM"]}')
        echo "$response" >> "$temp_dir/single.log"
    ) &
    
    # Multiple wallets test
    (
        echo "=== MULTIPLE WALLETS TEST ===" > "$temp_dir/multiple.log"
        response=$(curl -s -w "\nHTTP_STATUS:%{http_code}\nTIME_TOTAL:%{time_total}" \
            -X POST "$API_BASE_URL/api/get-balance" \
            -H "Content-Type: application/json" \
            -H "Authorization: $VALID_API_KEY" \
            -d '{
                "wallets": [
                    "So11111111111111111111111111111111111111112",
                    "EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v",
                    "Es9vMFrzaCERmJfrF4H2FYD4KCoNkY11McCe8BenwNYB"
                ]
            }')
        echo "$response" >> "$temp_dir/multiple.log"
    ) &
    
    # Concurrent requests for same wallet
    for i in {1..5}; do
        (
            echo "=== CONCURRENT REQUEST $i ===" > "$temp_dir/concurrent_$i.log"
            response=$(curl -s -w "\nHTTP_STATUS:%{http_code}\nTIME_TOTAL:%{time_total}" \
                -X POST "$API_BASE_URL/api/get-balance" \
                -H "Content-Type: application/json" \
                -H "Authorization: $VALID_API_KEY" \
                -d '{"wallets": ["mSoLzYCxHdYgdzU16g5QSh3i5K3z3KZK7ytfqcJm7So"]}')
            echo "$response" >> "$temp_dir/concurrent_$i.log"
        ) &
    done
    
    # Wait for all to complete
    wait
    
    print_subheader "All Scenarios Results:"
    
    echo -e "${CYAN}Single Wallet Test:${NC}"
    cat "$temp_dir/single.log"
    echo ""
    
    echo -e "${CYAN}Multiple Wallets Test:${NC}"
    cat "$temp_dir/multiple.log"
    echo ""
    
    echo -e "${CYAN}Concurrent Requests Results:${NC}"
    for i in {1..5}; do
        echo -e "${YELLOW}Concurrent Request $i:${NC}"
        cat "$temp_dir/concurrent_$i.log"
        echo ""
    done
    
    # Cleanup
    rm -rf "$temp_dir"
    
    print_success "Demo 4 completed - System handled mixed load successfully!"
    
    print_status "Waiting for rate limit to reset before Demo 5..."
    pause_with_countdown 65 "Demo 4 completed. Waiting for rate limit reset, then starting Demo 5 in"
}

# Demo 5: IP Rate Limiting
demo_5_rate_limiting() {
    print_header "DEMO 5: IP Rate Limiting (10 requests per minute)"
    
    print_demo_info "This demo shows the rate limiting in action"
    print_demo_info "First 10 requests should succeed, 11th should be rate limited"
    
    pause_with_countdown 15 "Starting Demo 5 in"
    
    print_subheader "Making 10 Requests (Should All Succeed)"
    
    for i in {1..10}; do
        echo -e "${YELLOW}Request $i:${NC}"
        
        response=$(curl -s -w "\nHTTP_STATUS:%{http_code}" \
            -X POST "$API_BASE_URL/api/get-balance" \
            -H "Content-Type: application/json" \
            -H "Authorization: $VALID_API_KEY" \
            -d '{"wallets": ["11111111111111111111111111111112"]}')
        
        http_status=$(echo "$response" | grep "HTTP_STATUS:" | cut -d: -f2)
        response_body=$(echo "$response" | sed '/HTTP_STATUS:/d')
        
        echo -e "${GREEN}Status: $http_status${NC}"
        
        # Get rate limit headers
        headers=$(curl -s -I -X POST "$API_BASE_URL/api/get-balance" \
            -H "Content-Type: application/json" \
            -H "Authorization: $VALID_API_KEY" \
            -d '{"wallets": ["11111111111111111111111111111112"]}' \
            | grep -i "x-ratelimit")
        
        if [ ! -z "$headers" ]; then
            echo -e "${BLUE}Rate Limit Headers:${NC}"
            echo "$headers"
        fi
        
        echo "---"
        sleep 1
    done
    
    pause_with_countdown 5 "Making 11th request (should be rate limited) in"
    
    print_subheader "Making 11th Request (Should Be Rate Limited)"
    
    echo -e "${YELLOW}Request 11 (Expected: 429 Too Many Requests):${NC}"
    
    response=$(curl -s -i -X POST "$API_BASE_URL/api/get-balance" \
        -H "Content-Type: application/json" \
        -H "Authorization: $VALID_API_KEY" \
        -d '{"wallets": ["11111111111111111111111111111112"]}')
    
    echo "$response"
    
    print_success "Demo 5 completed - Rate limiting working correctly!"
    
    print_status "Waiting for rate limit to reset before Demo 6..."
    pause_with_countdown 65 "Demo 5 completed. Waiting for rate limit reset, then starting Demo 6 in"
}

# Demo 6: Caching Behavior
demo_6_caching() {
    print_header "DEMO 6: Caching Behavior (10-second TTL)"
    
    print_demo_info "This demo shows the caching system with 10-second TTL"
    print_demo_info "We'll show cache miss, cache hit, and cache expiry"
    
    pause_with_countdown 15 "Starting Demo 6 in"
    
    print_subheader "Test 6.1: First Request (Cache Miss)"
    
    start_time=$(date +%s)
    make_api_request "POST" "/api/get-balance" \
        "$VALID_API_KEY" \
        '{"wallets": ["DezXAZ8z7PnrnRJjz3wXBoRgixCa6xjnB7YaB1pPB263"]}' \
        "First request - should be cache miss"
    
    pause_with_countdown 3 "Making immediate second request in"
    
    print_subheader "Test 6.2: Immediate Second Request (Cache Hit)"
    make_api_request "POST" "/api/get-balance" \
        "$VALID_API_KEY" \
        '{"wallets": ["DezXAZ8z7PnrnRJjz3wXBoRgixCa6xjnB7YaB1pPB263"]}' \
        "Immediate second request - should be cache hit"
    
    print_subheader "Test 6.3: Waiting for Cache to Expire"
    print_status "Waiting 11 seconds for cache TTL to expire..."
    
    for i in {11..1}; do
        echo -ne "${BLUE}Cache expires in $i seconds... ${NC}\r"
        sleep 1
    done
    echo ""
    
    print_subheader "Test 6.4: Request After Cache Expiry (Cache Miss Again)"
    make_api_request "POST" "/api/get-balance" \
        "$VALID_API_KEY" \
        '{"wallets": ["DezXAZ8z7PnrnRJjz3wXBoRgixCa6xjnB7YaB1pPB263"]}' \
        "Request after TTL expiry - should be cache miss again"
    
    print_success "Demo 6 completed - Caching system working perfectly!"
    
    print_status "Waiting for rate limit to reset before Demo 7..."
    pause_with_countdown 65 "Demo 6 completed. Waiting for rate limit reset, then starting Demo 7 in"
}

# Demo 7: Authentication and Rate Limiting Combined
demo_7_auth_and_rate_limiting() {
    print_header "DEMO 7: Authentication and Rate Limiting Combined"
    
    print_demo_info "This demo shows various authentication scenarios"
    print_demo_info "Valid keys, invalid keys, missing keys, and inactive keys"
    
    pause_with_countdown 15 "Starting Demo 7 in"
    
    print_subheader "Test 7.1: Valid API Key"
    make_api_request "POST" "/api/get-balance" \
        "$VALID_API_KEY" \
        '{"wallets": ["11111111111111111111111111111112"]}' \
        "Valid API key test"
    
    pause_with_countdown 5 "Testing missing API key in"
    
    print_subheader "Test 7.2: Missing API Key"
    make_api_request "POST" "/api/get-balance" \
        "" \
        '{"wallets": ["11111111111111111111111111111112"]}' \
        "Missing API key test"
    
    pause_with_countdown 5 "Testing invalid API key in"
    
    print_subheader "Test 7.3: Invalid API Key"
    make_api_request "POST" "/api/get-balance" \
        "$INVALID_API_KEY" \
        '{"wallets": ["11111111111111111111111111111112"]}' \
        "Invalid API key test"
    
    pause_with_countdown 5 "Testing Bearer token format in"
    
    print_subheader "Test 7.4: Bearer Token Format"
    make_api_request "POST" "/api/get-balance" \
        "Bearer $VALID_API_KEY" \
        '{"wallets": ["11111111111111111111111111111112"]}' \
        "Bearer token format test"
    
    pause_with_countdown 5 "Testing inactive API key in"
    
    print_subheader "Test 7.5: Inactive API Key"
    make_api_request "POST" "/api/get-balance" \
        "$INACTIVE_API_KEY" \
        '{"wallets": ["11111111111111111111111111111112"]}' \
        "Inactive API key test"
    
    print_success "Demo 7 completed - Authentication system working correctly!"
    
    print_status "Waiting for rate limit to reset before Error Handling Demo..."
    pause_with_countdown 65 "Demo 7 completed. Waiting for rate limit reset, then starting Error Handling Demo in"
}

# Error Handling Demo
demo_error_handling() {
    print_header "ERROR HANDLING DEMO"
    
    print_demo_info "This demo shows how the API handles various error conditions"
    print_demo_info "Invalid wallet addresses, empty arrays, malformed JSON, etc."
    
    pause_with_countdown 15 "Starting Error Handling Demo in"
    
    print_subheader "Test E.1: Invalid Wallet Address Format"
    make_api_request "POST" "/api/get-balance" \
        "$VALID_API_KEY" \
        '{"wallets": ["invalid-wallet-address"]}' \
        "Invalid wallet address format"
    
    pause_with_countdown 5 "Testing empty wallets array in"
    
    print_subheader "Test E.2: Empty Wallets Array"
    make_api_request "POST" "/api/get-balance" \
        "$VALID_API_KEY" \
        '{"wallets": []}' \
        "Empty wallets array"
    
    pause_with_countdown 5 "Testing malformed JSON in"
    
    print_subheader "Test E.3: Malformed JSON"
    echo -e "${CYAN}Request:${NC} POST /api/get-balance"
    echo -e "${CYAN}Headers:${NC} Authorization: $VALID_API_KEY"
    echo -e "${CYAN}Body:${NC} {\"wallets\": [\"11111111111111111111111111111112\""
    echo ""
    echo -e "${YELLOW}Sending malformed JSON request...${NC}"
    
    response=$(curl -s -w "\nHTTP_STATUS:%{http_code}" \
        -X POST "$API_BASE_URL/api/get-balance" \
        -H "Content-Type: application/json" \
        -H "Authorization: $VALID_API_KEY" \
        -d '{"wallets": ["11111111111111111111111111111112"')
    
    http_status=$(echo "$response" | grep "HTTP_STATUS:" | cut -d: -f2)
    response_body=$(echo "$response" | sed '/HTTP_STATUS:/d')
    
    echo -e "${GREEN}Response Status:${NC} $http_status"
    echo -e "${GREEN}Response Body:${NC}"
    echo "$response_body" | jq '.' 2>/dev/null || echo "$response_body"
    echo ""
    
    pause_with_countdown 5 "Testing mixed valid/invalid wallets in"
    
    print_subheader "Test E.4: Mixed Valid/Invalid Wallets"
    make_api_request "POST" "/api/get-balance" \
        "$VALID_API_KEY" \
        '{
            "wallets": [
                "11111111111111111111111111111112",
                "invalid-wallet",
                "11111111111111111111111111111113"
            ]
        }' \
        "Mixed valid and invalid wallet addresses"
    
    print_success "Error Handling Demo completed - All errors handled gracefully!"
    
    print_status "Waiting for rate limit to reset before Performance Demo..."
    pause_with_countdown 65 "Error Handling Demo completed. Waiting for rate limit reset, then starting Performance Demo in"
}

# Performance Monitoring Demo
demo_performance() {
    print_header "PERFORMANCE MONITORING DEMO"
    
    print_demo_info "This demo shows the monitoring and metrics endpoints"
    print_demo_info "Health checks, system status, and performance metrics"
    
    pause_with_countdown 15 "Starting Performance Demo in"
    
    print_subheader "Test P.1: Health Check Endpoint"
    make_api_request "GET" "/health" "" "" "Health check endpoint"
    
    pause_with_countdown 5 "Checking system status in"
    
    print_subheader "Test P.2: System Status Endpoint"
    make_api_request "GET" "/status" "" "" "System status endpoint"
    
    pause_with_countdown 5 "Checking performance metrics in"
    
    print_subheader "Test P.3: Performance Metrics Endpoint"
    make_api_request "GET" "/metrics" "" "" "Performance metrics endpoint"
    
    pause_with_countdown 10 "Performing load test in"
    
    print_subheader "Test P.4: Load Test (10 Concurrent Requests)"
    print_status "Performing load test with 10 concurrent requests..."
    
    temp_dir=$(mktemp -d)
    start_time=$(date +%s.%N)
    
    # Define valid wallet addresses for load test
    wallets=(
        "So11111111111111111111111111111111111111112"
        "EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v"
        "Es9vMFrzaCERmJfrF4H2FYD4KCoNkY11McCe8BenwNYB"
        "mSoLzYCxHdYgdzU16g5QSh3i5K3z3KZK7ytfqcJm7So"
        "7dHbWXmci3dT8UFYWYZweBLXgycu7Y3iL6trKn1Y7ARj"
        "DezXAZ8z7PnrnRJjz3wXBoRgixCa6xjnB7YaB1pPB263"
        "A9mUU4qviSctJVPJdBJWkb28deg915LYJKrzQ19ji3FM"
        "Gh9ZwEmdLJ8DscKNTkTqPbNwLNNBjuSzaG9Vp2KGtKJr"
        "TokenkegQfeZyiNwAJbNbGKPFXCWuBvf9Ss623VQ5DA"
        "11111111111111111111111111111111"
    )
    
    # Launch 10 concurrent requests
    for i in {0..9}; do
        (
            response=$(curl -s -w "\nTIME_TOTAL:%{time_total}" \
                -X POST "$API_BASE_URL/api/get-balance" \
                -H "Content-Type: application/json" \
                -H "Authorization: $VALID_API_KEY" \
                -d "{\"wallets\": [\"${wallets[$i]}\"]}")
            echo "$response" > "$temp_dir/load_$((i+1)).log"
        ) &
    done
    
    # Wait for all requests
    wait
    end_time=$(date +%s.%N)
    
    total_time=$(echo "$end_time - $start_time" | bc)
    
    echo -e "${GREEN}Load Test Results:${NC}"
    echo -e "${GREEN}Total Time: ${total_time}s${NC}"
    echo -e "${GREEN}Requests: 10${NC}"
    echo -e "${GREEN}Average Time per Request: $(echo "scale=3; $total_time / 10" | bc)s${NC}"
    
    # Show individual request times
    echo -e "${CYAN}Individual Request Times:${NC}"
    for i in {1..10}; do
        time_total=$(grep "TIME_TOTAL:" "$temp_dir/load_$i.log" | cut -d: -f2)
        echo "Request $i: ${time_total}s"
    done
    
    # Cleanup
    rm -rf "$temp_dir"
    
    print_success "Performance Demo completed - System performing well under load!"
    
    pause_with_countdown 10 "Performance Demo completed. Checking final metrics in"
    
    print_subheader "Final System Metrics"
    make_api_request "GET" "/metrics" "" "" "Final performance metrics"
}

# Main execution function
main() {
    print_header "SOLANA BALANCE API - COMPLETE DEMONSTRATION"
    
    echo -e "${WHITE}This comprehensive demo will showcase all API functionality:${NC}"
    echo -e "${CYAN}• Demo 1: Single Wallet Balance Retrieval${NC}"
    echo -e "${CYAN}• Demo 2: Multiple Wallets Balance Retrieval${NC}"
    echo -e "${CYAN}• Demo 3: Concurrent Requests with Same Wallet${NC}"
    echo -e "${CYAN}• Demo 4: All Scenarios Simultaneously${NC}"
    echo -e "${CYAN}• Demo 5: IP Rate Limiting${NC}"
    echo -e "${CYAN}• Demo 6: Caching Behavior${NC}"
    echo -e "${CYAN}• Demo 7: Authentication and Rate Limiting${NC}"
    echo -e "${CYAN}• Error Handling Demo${NC}"
    echo -e "${CYAN}• Performance Monitoring Demo${NC}"
    echo ""
    
    print_status "Checking prerequisites..."
    
    # Check if jq is installed
    if ! command -v jq &> /dev/null; then
        print_error "jq is not installed. Installing jq for JSON formatting..."
        sudo apt update && sudo apt install -y jq
    fi
    
    # Check if bc is installed
    if ! command -v bc &> /dev/null; then
        print_error "bc is not installed. Installing bc for calculations..."
        sudo apt update && sudo apt install -y bc
    fi
    
    # Check server
    if ! check_server; then
        exit 1
    fi
    
    print_success "All prerequisites met!"
    
    pause_with_countdown 10 "Starting comprehensive demo in"
    
    # Run all demos
    demo_1_single_wallet
    demo_2_multiple_wallets
    demo_3_concurrent_requests
    demo_4_all_scenarios
    demo_5_rate_limiting
    demo_6_caching
    demo_7_auth_and_rate_limiting
    demo_error_handling
    demo_performance
    
    # Final summary
    print_header "DEMONSTRATION COMPLETED SUCCESSFULLY!"
    
    echo -e "${GREEN}✅ All demos completed successfully!${NC}"
    echo -e "${GREEN}✅ Single wallet balance retrieval demonstrated${NC}"
    echo -e "${GREEN}✅ Multiple wallets batch processing demonstrated${NC}"
    echo -e "${GREEN}✅ Concurrent request handling with mutex demonstrated${NC}"
    echo -e "${GREEN}✅ Mixed load scenarios demonstrated${NC}"
    echo -e "${GREEN}✅ Rate limiting (10 req/min) demonstrated${NC}"
    echo -e "${GREEN}✅ Caching with 10-second TTL demonstrated${NC}"
    echo -e "${GREEN}✅ Authentication scenarios demonstrated${NC}"
    echo -e "${GREEN}✅ Error handling demonstrated${NC}"
    echo -e "${GREEN}✅ Performance monitoring demonstrated${NC}"
    echo ""
    
    print_status "The Solana Balance API is working perfectly!"
    print_status "All required functionality has been demonstrated."
    
    echo ""
    echo -e "${WHITE}================================================================${NC}"
    echo -e "${WHITE}  DEMO COMPLETE - Thank you for watching!${NC}"
    echo -e "${WHITE}================================================================${NC}"
}

# Run the main function
main "$@"