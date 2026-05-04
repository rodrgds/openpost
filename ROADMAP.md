# OpenPost Roadmap

> Status: May 2026 — Prioritized feature list and technical milestones

OpenPost is a lightweight, self-hosted social media scheduler. This roadmap outlines the planned evolution of the project as a linear, prioritized list.

---

## 🚀 Upcoming Features

1.  **Per-Platform Media Overrides (Frontend)**  
    Add UI to the "Customize per platform" view in the composer to allow selecting different media attachments for different social accounts. (Backend is already implemented).

2.  **Enhanced Thread Management**  
    Add backend and frontend support for atomic updates to threads that are already scheduled or have failed, allowing for easier mass-edits of multi-post chains.

3.  **API Key Management**  
    Implement a scoped API key system (`api_keys` table + middleware) and a management UI in Settings to allow programmatic access to the OpenPost API.

4.  **Directus Integration**  
    Enable two-way sync with Directus CMS, allowing published posts and media to be automatically archived or managed within a Directus collection.

5.  **MCP Server**  
    Implement an official Model Context Protocol (MCP) server for OpenPost. This will allow AI agents (Claude, Cursor, etc.) to interact directly with your instance to list accounts, schedule posts, and upload media.

6.  **AI Writing Assistance (Genkit)**  
    Integrate Google's Genkit for structured AI workflows, supporting Gemini and OpenAI for tone adjustment, rewrites, and content brainstorming directly in the composer.

7.  **Analytics & Engagement Tracking**  
    Implement a background worker to periodically poll platform APIs for engagement metrics (likes, reposts, clicks) and display them in a new Analytics dashboard.

8.  **Active Session Management**  
    Add a security view in Settings to list and revoke active JWT-based login sessions across different devices.

9.  **Full Pagination for List Endpoints**  
    Update the `Posts` and `Jobs` list endpoints to support full cursor or offset-based pagination (currently only supports simple limits).

10. **Spanish Localization**  
    Complete the translation files (`es.json`) to provide full Spanish language support alongside English and Portuguese.

---

## 🛠️ Technical Debt & Polish

11. **Test Coverage Expansion**  
    Increase backend test coverage to >80% for critical publishing and authentication paths.

12. **Background Worker Robustness**  
    Improve error recovery and exponential backoff strategies in the custom SQLite-backed job worker.

13. **UI/UX Refinement**  
    Ongoing polish of the Svelte 5 composer and dashboard interactions to ensure a "native-feeling" experience.
