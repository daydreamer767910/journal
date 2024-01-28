Build Process
Dependencies
- docker
Getting Started
git clone [https://github.com/jellyfin/jellyfin-web.git](https://github.com/daydreamer767910/journal.git)
cd journal
make

Directory Structure
.
└── src
    ├── assets            # Static assets
    ├── handler           # Legacy page views and controllers 🧹
    ├── model             # Basic data model 🧹
    ├── router            # router
    ├── store             # database
    ├── templates         # HTML templates 🧹
    ├── test              # test cases
    └── utils             # Utility functions
