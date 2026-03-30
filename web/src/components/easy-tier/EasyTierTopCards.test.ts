// @vitest-environment jsdom
import { defineComponent } from 'vue'
import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import EasyTierTopCards from './EasyTierTopCards.vue'

const NAlertStub = defineComponent({
  name: 'NAlert',
  template: '<div><slot /></div>',
})

const NCardStub = defineComponent({
  name: 'NCard',
  template: '<div><slot /></div>',
})

const NSpaceStub = defineComponent({
  name: 'NSpace',
  template: '<div><slot /></div>',
})

const IntroStub = defineComponent({
  name: 'EasyTierIntroAlert',
  template: '<div data-test="intro-stub" />',
})

const ProfilesStub = defineComponent({
  name: 'EasyTierProfilesCard',
  emits: ['create', 'delete', 'openRuntime', 'profileChange', 'update:activeProfileId', 'update:newProfileName'],
  template: `
    <div>
      <button data-test="create" @click="$emit('create')">create</button>
      <button data-test="delete" @click="$emit('delete')">delete</button>
      <button data-test="open-runtime" @click="$emit('openRuntime')">open</button>
      <button data-test="change-profile" @click="$emit('profileChange', 'p2')">change</button>
      <button data-test="set-active" @click="$emit('update:activeProfileId', 'p2')">setActive</button>
      <button data-test="set-name" @click="$emit('update:newProfileName', 'demo')">setName</button>
    </div>
  `,
})

describe('EasyTierTopCards', () => {
  it('forwards events from switch and profiles card', async () => {
    const wrapper = mount(EasyTierTopCards, {
      props: {
        easytierHostSupported: true,
        easytierAutostart: false,
        easytierAutostartSaving: false,
        activeProfileId: 'p1',
        newProfileName: '',
        profileOptions: [{ label: 'p1', value: 'p1' }],
        canDelete: true,
      },
      global: {
        stubs: {
          NAlert: NAlertStub,
          NCard: NCardStub,
          NSpace: NSpaceStub,
          EasyTierIntroAlert: IntroStub,
          EasyTierProfilesCard: ProfilesStub,
        },
      },
    })

    await wrapper.get('[role="switch"]').trigger('click')
    await wrapper.get('[data-test="create"]').trigger('click')
    await wrapper.get('[data-test="delete"]').trigger('click')
    await wrapper.get('[data-test="open-runtime"]').trigger('click')
    await wrapper.get('[data-test="change-profile"]').trigger('click')
    await wrapper.get('[data-test="set-active"]').trigger('click')
    await wrapper.get('[data-test="set-name"]').trigger('click')

    expect(wrapper.emitted('toggleAutostart')?.length).toBe(1)
    expect(wrapper.emitted('createProfile')?.length).toBe(1)
    expect(wrapper.emitted('deleteProfile')?.length).toBe(1)
    expect(wrapper.emitted('openRuntime')?.length).toBe(1)
    expect(wrapper.emitted('profileChange')?.[0]).toEqual(['p2'])
    expect(wrapper.emitted('update:activeProfileId')?.[0]).toEqual(['p2'])
    expect(wrapper.emitted('update:newProfileName')?.[0]).toEqual(['demo'])
  })
})
