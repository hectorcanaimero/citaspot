import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent, waitFor, act } from '@testing-library/react';
import React from 'react';
import { APIError } from '@/lib/api';

// ── Mocks ─────────────────────────────────────────────────────────────────────

// vi.hoisted() permite que la variable esté disponible dentro del factory de vi.mock,
// que se eleva al inicio del archivo antes de cualquier declaración.
const mockKnowledge = vi.hoisted(() => ({
  list:   vi.fn(),
  create: vi.fn(),
  update: vi.fn(),
  remove: vi.fn(),
}));

vi.mock('next/navigation', () => ({
  usePathname: vi.fn(() => '/dashboard/knowledge'),
  useRouter:   vi.fn(() => ({ push: vi.fn(), refresh: vi.fn() })),
}));

vi.mock('@/lib/supabase/browser', () => ({
  createClient: vi.fn(() => ({
    auth: { getSession: vi.fn(), signOut: vi.fn() },
  })),
}));

vi.mock('@/lib/api', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@/lib/api')>();
  return {
    ...actual,
    knowledge: mockKnowledge,
  };
});

// ── Datos de prueba ───────────────────────────────────────────────────────────

const DOC_1 = {
  id: 'doc-1',
  tenant_id: 'tenant-1',
  category: 'faq',
  title: 'Horarios de atención',
  content: 'Abrimos de lunes a viernes de 9am a 6pm',
  is_active: true,
  created_at: '2026-01-01T00:00:00Z',
  updated_at: '2026-01-01T00:00:00Z',
};

const DOC_2 = {
  id: 'doc-2',
  tenant_id: 'tenant-1',
  category: 'services',
  title: 'Corte de cabello',
  content: 'Servicio de corte de cabello para hombres y mujeres desde $15',
  is_active: false,
  created_at: '2026-01-02T00:00:00Z',
  updated_at: '2026-01-02T00:00:00Z',
};

// ── Import del componente ─────────────────────────────────────────────────────

import KnowledgePage from '@/app/dashboard/knowledge/page';

// ── Tests ─────────────────────────────────────────────────────────────────────

describe('KnowledgePage', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    // window.confirm: retorna true por defecto en tests de delete
    vi.stubGlobal('confirm', vi.fn(() => true));
  });

  // ── TESTS QUE DEBEN FALLAR (exponen bugs reales) ───────────────────────────

  describe('Bug: null data del API Go (slice nil → JSON null)', () => {
    it('DEBE mostrar estado vacío cuando knowledge.list() retorna { data: null }', async () => {
      // El Go API serializa un slice nil como null en JSON.
      // La respuesta llega: { "data": null }
      // El componente hace setDocs(null) → docs.map() → TypeError crash
      mockKnowledge.list.mockResolvedValue({ data: null });

      render(<KnowledgePage />);

      // Este test FALLA sin el fix: el componente crashea antes de llegar aquí.
      await waitFor(() => {
        expect(
          screen.getByText(/no hay documentos/i)
        ).toBeInTheDocument();
      });
    });

    it('NO debe lanzar TypeError cuando data es null', async () => {
      mockKnowledge.list.mockResolvedValue({ data: null });

      expect(() => render(<KnowledgePage />)).not.toThrow();

      // Esperar a que termine el efecto asíncrono — sin crash
      await act(async () => {
        await new Promise(r => setTimeout(r, 50));
      });
    });
  });

  // ── TESTS NORMALES (estado base) ───────────────────────────────────────────

  describe('Estado de carga', () => {
    it('muestra spinner mientras carga', () => {
      // Promesa que nunca resuelve → se queda en loading
      mockKnowledge.list.mockReturnValue(new Promise(() => {}));
      render(<KnowledgePage />);
      expect(document.querySelector('svg.animate-spin')).toBeInTheDocument();
    });
  });

  describe('Lista de documentos', () => {
    beforeEach(() => {
      mockKnowledge.list.mockResolvedValue({ data: [DOC_1, DOC_2] });
    });

    it('muestra los documentos cargados', async () => {
      render(<KnowledgePage />);
      await screen.findByText('Horarios de atención');
      expect(screen.getByText('Corte de cabello')).toBeInTheDocument();
    });

    it('muestra la categoría del documento como badge', async () => {
      render(<KnowledgePage />);
      await screen.findByText('Preguntas frecuentes'); // label de 'faq'
      expect(screen.getByText('Servicios')).toBeInTheDocument(); // label de 'services'
    });

    it('muestra badge "Inactivo" para documentos inactivos', async () => {
      render(<KnowledgePage />);
      await screen.findByText('Corte de cabello');
      expect(screen.getByText('Inactivo')).toBeInTheDocument();
    });

    it('no muestra badge "Inactivo" para documentos activos', async () => {
      mockKnowledge.list.mockResolvedValue({ data: [DOC_1] }); // solo el activo
      render(<KnowledgePage />);
      await screen.findByText('Horarios de atención');
      expect(screen.queryByText('Inactivo')).not.toBeInTheDocument();
    });
  });

  describe('Estado vacío', () => {
    it('muestra estado vacío cuando data es [] (sin documentos)', async () => {
      mockKnowledge.list.mockResolvedValue({ data: [] });
      render(<KnowledgePage />);
      await screen.findByText(/no hay documentos/i);
      expect(screen.getByText(/crear primer documento/i)).toBeInTheDocument();
    });
  });

  describe('Error en carga', () => {
    it('muestra mensaje de error cuando list() falla', async () => {
      mockKnowledge.list.mockRejectedValue(new Error('Network Error'));
      render(<KnowledgePage />);
      await screen.findByText('No se pudo cargar la base de conocimiento');
    });
  });

  // ── Formulario de creación ──────────────────────────────────────────────────

  describe('Crear documento', () => {
    beforeEach(() => {
      mockKnowledge.list.mockResolvedValue({ data: [] });
    });

    it('abre el formulario al hacer click en "Nuevo documento"', async () => {
      render(<KnowledgePage />);
      await screen.findByText(/no hay documentos/i);
      fireEvent.click(screen.getAllByRole('button', { name: /nuevo documento/i })[0]);
      // El formulario muestra un heading "Nuevo documento" (distinto al botón)
      expect(screen.getByRole('heading', { name: 'Nuevo documento' })).toBeInTheDocument();
    });

    it('llama a knowledge.create con los datos del formulario', async () => {
      const newDoc = { ...DOC_1, id: 'doc-new' };
      mockKnowledge.create.mockResolvedValue(newDoc);

      render(<KnowledgePage />);
      await screen.findByText(/no hay documentos/i);

      // Abrir formulario
      fireEvent.click(screen.getAllByRole('button', { name: /nuevo documento/i })[0]);

      // Llenar título (el campo tiene label "Título")
      const titleInput = screen.getByLabelText('Título');
      fireEvent.change(titleInput, { target: { value: 'Horarios de atención' } });

      // Llenar contenido
      const contentArea = screen.getByPlaceholderText(/describe esta información/i);
      fireEvent.change(contentArea, { target: { value: 'Abrimos de lunes a viernes de 9am a 6pm' } });

      // Enviar
      fireEvent.click(screen.getByRole('button', { name: /crear documento/i }));

      await waitFor(() => {
        expect(mockKnowledge.create).toHaveBeenCalledOnce();
        expect(mockKnowledge.create).toHaveBeenCalledWith(
          expect.objectContaining({ title: 'Horarios de atención', category: 'faq' })
        );
      });
    });

    it('agrega el documento a la lista después de crear', async () => {
      const newDoc = { ...DOC_1, id: 'doc-new' };
      mockKnowledge.create.mockResolvedValue(newDoc);
      mockKnowledge.list.mockResolvedValue({ data: [] });

      render(<KnowledgePage />);
      await screen.findByText(/no hay documentos/i);

      fireEvent.click(screen.getAllByRole('button', { name: /nuevo documento/i })[0]);
      fireEvent.change(screen.getByLabelText('Título'), { target: { value: 'Horarios de atención' } });
      fireEvent.change(screen.getByPlaceholderText(/describe esta información/i), {
        target: { value: 'Abrimos de lunes a viernes de 9am a 6pm' },
      });
      fireEvent.click(screen.getByRole('button', { name: /crear documento/i }));

      await waitFor(() => {
        expect(screen.getByText('Horarios de atención')).toBeInTheDocument();
      });
    });

    it('muestra error si knowledge.create() falla con APIError', async () => {
      mockKnowledge.create.mockRejectedValue(new APIError(422, 'El contenido debe tener al menos 10 caracteres'));

      render(<KnowledgePage />);
      await screen.findByText(/no hay documentos/i);

      fireEvent.click(screen.getAllByRole('button', { name: /nuevo documento/i })[0]);
      fireEvent.change(screen.getByLabelText('Título'), { target: { value: 'Test' } });
      fireEvent.change(screen.getByPlaceholderText(/describe esta información/i), { target: { value: 'corto' } });
      fireEvent.click(screen.getByRole('button', { name: /crear documento/i }));

      // El mismo estado 'error' aparece en el banner de página Y en el <p> del form.
      // Aquí verificamos el error inline del form (selector: 'p').
      await screen.findByText('El contenido debe tener al menos 10 caracteres', { selector: 'p' });
    });
  });

  // ── Formulario de edición ───────────────────────────────────────────────────

  describe('Editar documento', () => {
    beforeEach(() => {
      mockKnowledge.list.mockResolvedValue({ data: [DOC_1] });
    });

    it('prefija el formulario con los datos del documento', async () => {
      render(<KnowledgePage />);
      await screen.findByText('Horarios de atención');

      // El botón de editar ahora tiene aria-label="Editar"
      fireEvent.click(screen.getByRole('button', { name: 'Editar' }));

      // Verificar que el título está prellenado
      const titleInput = screen.getByLabelText('Título') as HTMLInputElement;
      expect(titleInput.value).toBe('Horarios de atención');
    });

    it('llama a knowledge.update con el ID correcto al guardar', async () => {
      const updated = { ...DOC_1, title: 'Horarios actualizados' };
      mockKnowledge.update.mockResolvedValue(updated);

      render(<KnowledgePage />);
      await screen.findByText('Horarios de atención');

      fireEvent.click(screen.getByRole('button', { name: 'Editar' }));

      const titleInput = screen.getByLabelText('Título') as HTMLInputElement;
      fireEvent.change(titleInput, { target: { value: 'Horarios actualizados' } });

      fireEvent.click(screen.getByRole('button', { name: /guardar cambios/i }));

      await waitFor(() => {
        expect(mockKnowledge.update).toHaveBeenCalledWith(
          'doc-1',
          expect.objectContaining({ title: 'Horarios actualizados' })
        );
      });
    });
  });

  // ── Eliminar documento ──────────────────────────────────────────────────────

  describe('Eliminar documento', () => {
    beforeEach(() => {
      mockKnowledge.list.mockResolvedValue({ data: [DOC_1] });
      mockKnowledge.remove.mockResolvedValue(undefined);
    });

    it('llama a knowledge.remove con el ID cuando se confirma', async () => {
      render(<KnowledgePage />);
      await screen.findByText('Horarios de atención');

      // El botón de eliminar ahora tiene aria-label="Eliminar"
      fireEvent.click(screen.getByRole('button', { name: 'Eliminar' }));

      await waitFor(() => {
        expect(mockKnowledge.remove).toHaveBeenCalledWith('doc-1');
      });
    });

    it('elimina el documento de la lista tras borrar', async () => {
      render(<KnowledgePage />);
      await screen.findByText('Horarios de atención');

      fireEvent.click(screen.getByRole('button', { name: 'Eliminar' }));

      await waitFor(() => {
        expect(screen.queryByText('Horarios de atención')).not.toBeInTheDocument();
      });
    });

    it('NO llama a knowledge.remove si el usuario cancela', async () => {
      vi.stubGlobal('confirm', vi.fn(() => false));
      render(<KnowledgePage />);
      await screen.findByText('Horarios de atención');

      fireEvent.click(screen.getByRole('button', { name: 'Eliminar' }));

      expect(mockKnowledge.remove).not.toHaveBeenCalled();
    });
  });
});
