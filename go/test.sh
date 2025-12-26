#!/usr/bin/env bash

# © 2025 Sharon Aicler (saichler@gmail.com)
#
# Layer 8 Ecosystem is licensed under the Apache License, Version 2.0.
# You may obtain a copy of the License at:
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# Fail on errors and don't open cover file
set -e
# clean up
rm -rf go.mod
rm -rf go.sum
rm -rf vendor

# fetch dependencies
go mod init
GOPROXY=direct GOPRIVATE=github.com go mod tidy
go mod vendor

echo "******************************************************"
echo "* Make sure you built security before running this tests"
echo "* Shallow Security exist in https://github.com/saichler/l8utils/tree/main/go/utils/shallow_security/build.sh"
echo "******************************************************"
read -n 1 -s -r -p "Press any key to continue..."

# Run unit tests with coverage
go test -tags=unit -v -coverpkg=./collector/... -coverprofile=cover.html ./... --failfast

#rm -rf ./tests/loader.so

# Open the coverage report in a browser
go tool cover -html=cover.html
