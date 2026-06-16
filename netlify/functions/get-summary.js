exports.handler = async function(event, context) {
    if (event.httpMethod !== "GET") {
        return { statusCode: 405, body: "Only GET allowed" };
    }

    // Notice this points to your new /summary route!
    const googleUrl = "https://transaction-api-1036329277666.us-central1.run.app/summary";

    try {
        const response = await fetch(googleUrl, {
            method: "GET",
            headers: {
                "X-API-Key": process.env.API_PASSWORD 
            }
        });

        const data = await response.text();

        return {
            statusCode: response.status,
            body: data
        };
    } catch (error) {
        return { statusCode: 500, body: "Internal Server Error" };
    }
};