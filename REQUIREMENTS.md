Temporal Coding Challenge

This challenge is designed to help us understand your experience with Temporal.
We expect you to spend approximately 4–6 hours completing it.

During the interview, you will be asked to demonstrate your solution and walk us through your approach.

Please upload your code to an accessible repository (e.g., GitHub) for review.

You are welcome to use AI tools to accelerate your work.

📦 Scenario – Order Process Workflow

You are building a simplified Order Processing System using Temporal.
The system should:

Accept an order request

Validate the order by calling a mock external service (e.g., WireMock)

Process the order asynchronously using Temporal workflows & activities

🧱 Basic Requirements (Core Temporal Concepts)
1. Workflow & Activities

Implement a main workflow that:

Accepts an order request, e.g.:

Order {
id: string,
items: string[],
amount: number
}


Calls an activity to validate the order by making an HTTP request to a mock server (WireMock)

Calls another activity to process the order (simulate business logic)

Ensure proper:

Retry policies

Timeouts for activities

2. Worker Setup

Create a Temporal worker that:

Registers workflow and activities

Includes a simple starter to trigger the workflow

3. Mock Server

Configure WireMock (or similar) to simulate an external validation API.

Example endpoint:

POST /validate


The mock server should return success or failure based on the order.amount.

4. Signals & Queries

Add:

A signal to update the order status (e.g., cancel, expedite)

A query to check the current workflow state

5. Unit Tests

Write unit tests for at least one activity using mocking (e.g., mock HTTP client).

⭐ Advanced Challenge (Bonus Points)
1. Encryption / Decryption

Implement a Payload Codec to encrypt/decrypt workflow inputs & outputs.

2. Child Workflow

Add an optional child workflow for payment processing.

3. Versioning

Demonstrate use of:

Workflow.getVersion


to add a new step while maintaining backward compatibility.

4. Messaging

Use:

Signals to trigger expedited processing

Queries to fetch workflow progress