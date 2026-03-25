export type BooleanOption = {
  label: string
  value: boolean
}

export const ENABLED_OPTIONS: BooleanOption[] = [
  { label: '启用', value: true },
  { label: '停用', value: false },
]

export const READ_ONLY_OPTIONS: BooleanOption[] = [
  { label: '读写', value: false },
  { label: '只读', value: true },
]

