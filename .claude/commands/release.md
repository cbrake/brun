- update the changelog with the version number and the current date
- update the README.md install section example install on Linux and replace with
  current version
- git commit, push
- tag the repo with the next version number
- run `goreleaser release --clean --release-notes .release-notes.md`
