# Threads

Threads let you publish multi-post sequences in order.

## How they work

- Each child post references its place in the chain.
- OpenPost publishes sequentially rather than firing every item at once.
- Provider behavior differs underneath the same OpenPost concept.

## Caveats

- LinkedIn uses comment-style replies on the first post.
- Failures in early posts can block later posts in the same thread.
