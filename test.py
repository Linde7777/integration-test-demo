#!/usr/bin/env python3
import subprocess
import sys


def run_command(command):
    process = subprocess.Popen(command, stdout=subprocess.PIPE, stderr=subprocess.PIPE, shell=True)
    output, error = process.communicate()
    return output.decode('utf-8'), error.decode('utf-8'), process.returncode


def go_test_coverage():
    print("Running tests with coverage...")
    output, error, code = run_command("go test -cover ./...")
    print(output)
    if code != 0:
        print(f"Error: {error}")
        sys.exit(code)


def generate_coverage_profile():
    print("Generating coverage profile...")
    output, error, code = run_command("go test -coverprofile=coverage.out ./...")
    if code != 0:
        print(f"Error: {error}")
        sys.exit(code)


def view_coverage_report():
    print("Viewing coverage report...")
    output, error, code = run_command("go tool cover -func=coverage.out")
    print(output)
    if code != 0:
        print(f"Error: {error}")
        sys.exit(code)


def generate_html_report():
    print("Generating HTML coverage report...")
    output, error, code = run_command("go tool cover -html=coverage.out -o coverage.html")
    if code != 0:
        print(f"Error: {error}")
    else:
        print("HTML report generated: coverage.html")


def check_coverage_threshold(threshold=80.0):
    print(f"Checking if coverage meets {threshold}% threshold...")
    command = f"go test -cover -coverprofile=coverage.out ./... && go tool cover -func=coverage.out | grep total | awk '{{if ($3 < {threshold}) exit 1}}'"
    output, error, code = run_command(command)
    if code != 0:
        print(f"Coverage is below {threshold}%")
        sys.exit(code)
    else:
        print(f"Coverage meets or exceeds {threshold}%")


if __name__ == "__main__":
    go_test_coverage()
    generate_coverage_profile()
    view_coverage_report()
    generate_html_report()
    check_coverage_threshold(80.0)
