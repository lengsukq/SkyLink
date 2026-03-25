import type { InjectionKey, Ref } from 'vue'

export type CfAccount = {
  id: number
  name?: string
  zone_id?: number | string
}

export const cfCurrentAccountIdKey: InjectionKey<Ref<number | null>> = Symbol('cfCurrentAccountId')
export const cfAccountsKey: InjectionKey<Ref<CfAccount[]>> = Symbol('cfAccounts')
export const refreshCfStateKey: InjectionKey<() => Promise<void>> = Symbol('refreshCfState')

