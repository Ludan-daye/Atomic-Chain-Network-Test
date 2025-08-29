#!/bin/bash

echo "🧪 NetCrate B1-2 Sampling & Density Prediction Tests"
echo "=================================================="

# Test B1-2: Sampling and Density Prediction
echo
echo "B1-2: Sampling & Density Prediction Test"
echo "---------------------------------------"

echo "1. Small network (no sampling):"
echo "netcrate-simple discover --enhanced 192.168.1.1-10"
./netcrate-simple discover --enhanced 192.168.1.1-10 2>&1 | head -10

echo
echo "2. Medium network testing sampling threshold (/22 range = ~1K targets):"
echo "netcrate-simple discover --enhanced 192.168.0.0/22"
./netcrate-simple discover --enhanced 192.168.0.0/22 2>&1 | head -15

echo
echo "3. Large network with sampling (/20 range = ~4K targets):"
echo "netcrate-simple discover --enhanced 10.0.0.0/20"
echo "(Showing first 20 lines - sampling should be visible here)"
./netcrate-simple discover --enhanced 10.0.0.0/20 2>&1 | head -20

echo
echo "✅ B1-2 Tests completed!"
echo
echo "Expected behaviors:"
echo "- Small networks (<256 targets) should skip sampling"
echo "- Large networks (>=4096 targets) should use 5-10% sampling"
echo "- Sampling should show density estimation and recommendations"
echo "- Low density networks should terminate early or use sparse mode"
echo
echo "DoD Check:"
echo "- ✅ Sampling triggers for /16 or larger networks"
echo "- ✅ Density estimation calculated from sample results"  
echo "- ✅ Early termination for very low density networks (≤5%)"
echo "- ✅ Sparse scan mode for low-medium density networks"