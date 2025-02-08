package routes

import (
	"fmt"
	"net/http"
)

// PrivacyPolicyHandler serves the Privacy Policy content
func PrivacyPolicyHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")

	// Serve Privacy Policy content as HTML
	html := `
	<!DOCTYPE html>
	<html lang="en">
	<head>
		<meta charset="UTF-8">
		<meta name="viewport" content="width=device-width, initial-scale=1.0">
		<title>Privacy Policy</title>
	</head>
	<body>
		<h1>Privacy Policy</h1>
		<p>Welcome to VibinConnect. This Privacy Policy outlines how we collect, use, and protect your data.</p>
		<p>We prioritize your privacy and ensure your data is handled responsibly.</p>
		<p>Contact us at <a href="mailto:support@vibinconnect.com">support@vibinconnect.com</a> for questions.</p>
	</body>
	</html>
	`
	fmt.Fprint(w, html)
}
