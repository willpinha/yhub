<div align="center">

# yhub

Manage all your Git repositories through a single, centralized repository

[Motivation]() • [Installation]() • [Documentation]() • [Contributing]() • [License]()

</div>

# Motivation

Have you ever experienced one of the following situations?

- You have Git repositories scattered throughout your filesystem, and it takes some time to find a specific one
- You have more than one Git account configured on the same computer, and you had to define your own SSH key mapping logic in `.gitconfig`
- You and your team share a set of repositories, but each member needs to manually configure each repository locally
- You need to pass the repository paths to the AI ​​so it has context about which repository you are referring to
- You simply want an easier way to organize your Git repositories and maintain that organization across different computers

If you identified with any of these points, **yhub** might be for you. It's a tool for centralizing the organization of your repositories into a single Git repository. Some advantages this can bring you:

- **Portability:** Your repository organization is the same across different computers
- **AI Integration:** Instead of saying "Given the repository in `~/code/repos/my-repo`, do this", you say "Given `my-repo`, do this" and the AI ​​will know where to look for it
- **Team collaboration:** Your team can centralize the organization of all your repositories across team members
- **Simplicity:** You don't need to configure your `.gitconfig` file with SSH key mapping logic

# Installation

Download the archive for your platform from the [latest release](https://github.com/willpinha/yhub/releases/latest), extract it, and place the `yhub` binary somewhere in your `PATH`

Alternatively, if you have Go installed:

```sh
go install github.com/willpinha/yhub@latest
```

# Documentation

1. [Get started]()
2. [Configuration files]()
3. [Integration with AI]()
4. [Team collaboration]()

## Get started

## Configuration files

## Integration with AI

## Team collaboration

# Contributing

# License
