import {useRef, useState} from 'react'
import {Text, useInput} from 'ink'
import {stripPointerReports} from '../events/usePointer.js'
import {useAnpanTheme} from '../theme/palette.js'

type KeyFieldProps = {
  value: string
  onChange: (value: string) => void
  onSubmit?: (value: string) => void
  placeholder?: string
  width?: number
  history?: string[]
  onTab?: () => void
}

const wordLeft = (text: string, from: number) => {
  let i = from
  while (i > 0 && !/\w/.test(text[i - 1]!)) i--
  while (i > 0 && /\w/.test(text[i - 1]!)) i--
  return i
}

const wordRight = (text: string, from: number) => {
  let i = from
  while (i < text.length && !/\w/.test(text[i]!)) i++
  while (i < text.length && /\w/.test(text[i]!)) i++
  return i
}

export function KeyField({
  value,
  onChange,
  onSubmit,
  placeholder = '',
  width = 40,
  history = [],
  onTab,
}: KeyFieldProps) {
  const palette = useAnpanTheme()
  const [cursorState, setCursorState] = useState(value.length)
  const [anchorState, setAnchorState] = useState<number | null>(null)
  const [historyPos, setHistoryPos] = useState<number | null>(null)
  const draftRef = useRef('')
  const offsetRef = useRef(0)

  const cursor = Math.min(cursorState, value.length)
  const anchor = anchorState === null ? null : Math.min(anchorState, value.length)

  const selStart = anchor === null ? cursor : Math.min(anchor, cursor)
  const selEnd = anchor === null ? cursor : Math.max(anchor, cursor)
  const hasSelection = anchor !== null && anchor !== cursor

  const update = (newValue: string, newCursor: number) => {
    onChange(stripPointerReports(newValue))
    setCursorState(newCursor)
    setAnchorState(null)
    setHistoryPos(null)
  }

  const deleteSelection = () => {
    const next = value.slice(0, selStart) + value.slice(selEnd)
    update(next, selStart)
  }

  useInput((input, key) => {
    if (key.tab) {
      onTab?.()
      return
    }
    if (key.return) {
      onSubmit?.(value)
      return
    }
    if (key.escape) {
      if (value) update('', 0)
      return
    }

    if (key.upArrow && !key.shift && !key.meta) {
      if (history.length === 0) return
      const nextPos = historyPos === null ? 0 : Math.min(historyPos + 1, history.length - 1)
      if (historyPos === null) draftRef.current = value
      const entry = history[nextPos]!
      onChange(entry)
      setCursorState(entry.length)
      setAnchorState(null)
      setHistoryPos(nextPos)
      return
    }
    if (key.downArrow && !key.shift && !key.meta) {
      if (historyPos === null) return
      const nextPos = historyPos - 1
      if (nextPos < 0) {
        onChange(draftRef.current)
        setCursorState(draftRef.current.length)
        setAnchorState(null)
        setHistoryPos(null)
      } else {
        const entry = history[nextPos]!
        onChange(entry)
        setCursorState(entry.length)
        setAnchorState(null)
        setHistoryPos(nextPos)
      }
      return
    }

    if (key.leftArrow) {
      const to = key.meta ? wordLeft(value, cursor) : cursor - 1
      if (to >= 0) {
        if (key.shift) {
          if (anchor === null) setAnchorState(cursor)
          setCursorState(to)
        } else {
          setCursorState(hasSelection ? selStart : to)
          setAnchorState(null)
        }
      }
      return
    }
    if (key.rightArrow) {
      const to = key.meta ? wordRight(value, cursor) : cursor + 1
      if (to <= value.length) {
        if (key.shift) {
          if (anchor === null) setAnchorState(cursor)
          setCursorState(to)
        } else {
          setCursorState(hasSelection ? selEnd : to)
          setAnchorState(null)
        }
      }
      return
    }

    if (key.ctrl) {
      if (input === 'a') {setCursorState(0); setAnchorState(null); return}
      if (input === 'e') {setCursorState(value.length); setAnchorState(null); return}
      if (input === 'u') {update(value.slice(cursor), 0); return}
      if (input === 'k') {update(value.slice(0, cursor), cursor); return}
      if (input === 'w') {
        const to = wordLeft(value, cursor)
        update(value.slice(0, to) + value.slice(cursor), to)
        return
      }
      if (input === 'h') {
        if (hasSelection) {deleteSelection(); return}
        if (cursor > 0) update(value.slice(0, cursor - 1) + value.slice(cursor), cursor - 1)
        return
      }
    }

    if (key.backspace || key.delete) {
      if (hasSelection) {deleteSelection(); return}
      if (key.backspace && cursor > 0) {
        if (key.meta) {
          const to = wordLeft(value, cursor)
          update(value.slice(0, to) + value.slice(cursor), to)
        } else {
          update(value.slice(0, cursor - 1) + value.slice(cursor), cursor - 1)
        }
      } else if (key.delete && cursor < value.length) {
        update(value.slice(0, cursor) + value.slice(cursor + 1), cursor)
      }
      return
    }

    if (input && !key.ctrl && !key.meta) {
      const cleaned = stripPointerReports(input)
      if (!cleaned) return

      let next: string
      let newCur: number
      if (hasSelection) {
        next = value.slice(0, selStart) + cleaned + value.slice(selEnd)
        newCur = selStart + cleaned.length
      } else {
        next = value.slice(0, cursor) + cleaned + value.slice(cursor)
        newCur = cursor + cleaned.length
      }

      update(next, newCur)
    }
  })

  if (cursor < offsetRef.current) offsetRef.current = cursor
  if (cursor > offsetRef.current + width - 1) offsetRef.current = cursor - width + 1
  const offset = offsetRef.current

  const visible = value.slice(offset, offset + width) || placeholder.slice(0, width)
  const isPlaceholder = !value
  const cursorInView = cursor - offset

  const chars = [...(isPlaceholder ? placeholder.slice(0, width) : visible)]

  return (
    <Text>
      {chars.map((ch, i) => {
        const isCursor = !isPlaceholder && i === cursorInView
        const isSelected = !isPlaceholder && hasSelection && (i + offset) >= selStart && (i + offset) < selEnd
        if (isCursor) {
          return (
            <Text key={i} inverse color={palette.primary}>
              {ch}
            </Text>
          )
        }
        if (isSelected) {
          return (
            <Text key={i} inverse color={palette.primary} dimColor>
              {ch}
            </Text>
          )
        }
        return (
          <Text key={i} color={isPlaceholder ? palette.muted : palette.primary} dimColor={isPlaceholder && palette.dimAccent}>
            {ch}
          </Text>
        )
      })}
      {!isPlaceholder && cursorInView >= chars.length && (
        <Text inverse color={palette.primary}>
          {' '}
        </Text>
      )}
    </Text>
  )
}
