<script lang="ts">
  const icons = {
    lock: "\uf023",
    key: "\uf084",
    copy: "\uf0c5",
    check: "\uf00c",
    file: "\uf15b",
    download: "\uf019",
    trash: "\uf1f8",
  } as const;

  type IconName = keyof typeof icons;

  interface Props {
    name: IconName;
  }

  let { name }: Props = $props();
</script>

<span data-icon aria-hidden="true">{icons[name]}</span>

<style>
  [data-icon] {
    font-family: "Symbols Nerd Font";
    font-style: normal;
    font-weight: 400;
  }
</style>
