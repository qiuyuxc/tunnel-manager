import manifest from './navigation.json'

export const sections = manifest

export const pagePaths = ['index', ...sections.flatMap(section => section.groups.flatMap(group => group.pages.map(page => page.startsWith('/') ? page.slice(1) : `${section.path}/${page}`)))]
