#!/bin/bash

# Test script for interactive Quick mode
echo "Testing NetCrate Quick Mode Interactive Features"
echo "================================================"

echo
echo "1. Testing default (zero-config) mode:"
echo "netcrate-simple quick"
./netcrate-simple quick

echo
echo
echo "2. Testing fast profile selection:"
echo "Simulating: netcrate quick --interactive"
echo "Port Set: 1 (top100)"
echo "Speed Profile: 2 (fast)"
echo "2" | echo "1" | ./netcrate-simple quick --interactive

echo
echo
echo "3. Testing web ports + custom speed:"
echo "Simulating selection of web ports and custom speed"
# This would require more complex input simulation
echo "[Skipped - requires interactive input]"

echo
echo "✅ All basic tests completed!"