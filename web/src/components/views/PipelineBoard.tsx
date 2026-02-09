import { useParams } from '@tanstack/react-router'
import { PipelineBoard as Board } from '@/components/pipeline/PipelineBoard'

export function PipelineBoard() {
  const { objectType } = useParams({ strict: false }) as { objectType: string }
  return <Board objectType={objectType} />
}
