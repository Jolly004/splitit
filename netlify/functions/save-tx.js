exports.handler = async function(event, context) {
    if (event.httpMethod !== "POST") {
        return { statusCode: 405, body: "Only POST allowed" };
    }

    const googleUrl = "https://transaction-api-1036329277666.us-central1.run.app/transaction";

    try {
        // We forward the request to Google Cloud, secretly attaching the password here!
        const response = await fetch(googleUrl, {
            method: "POST",
            headers: {
                "Content-Type": "application/json",
                "X-API-Key": process.env.API_PASSWORD 
            },
            body: event.body
        });

        const resultText = await response.text();

        return {
            statusCode: response.status,
            body: resultText
        };
    } catch (error) {
        return { statusCode: 500, body: "Internal Server Error" };
    }
};