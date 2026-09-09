package tenant

import (
	"context"
	"fmt"
)

// MarkProvisioning, tenant'i 'provisioning' durumuna alir.
//
// Gecis yalnizca 'pending', 'failed' veya zaten 'provisioning' durumundan
// yapilir. Aktif bir tenant bu cagriyla geri cekilemez.
//
// NEDEN kaynak durum kosulu: bu yazim fencing korumasinin disindadir.
// Lease'ini kaybetmis eski bir worker, guncel worker tenant'i aktiflestirdikten
// sonra bu cagriyi yapabilir. Kosul olmasaydi calisan bir tenant'i kurulum
// durumuna dusururdu.
//
// Etkilenen satir olmamasi hata degildir: gecis uygulanmadi demektir ve bu
// beklenen bir durumdur.
//
// Karmasiklik: O(1), birincil anahtar uzerinden tekil guncelleme.
func (r *PostgresRepository) MarkProvisioning(ctx context.Context, tenantID string) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE tenants
		SET status = 'provisioning', updated_at = NOW()
		WHERE id = $1
		  AND deleted_at IS NULL
		  AND status IN ('pending', 'provisioning', 'failed')`, tenantID)
	if err != nil {
		return fmt.Errorf("could not move tenant to 'provisioning': %w", err)
	}

	return nil
}

// MarkFailed, tenant'i 'failed' durumuna alir.
//
// Aktif bir tenant basarisiz isaretlenmez: kurulum zaten tamamlanmistir ve
// gecikmis bir bildirim onu bozmamalidir.
func (r *PostgresRepository) MarkFailed(ctx context.Context, tenantID string) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE tenants
		SET status = 'failed', updated_at = NOW()
		WHERE id = $1
		  AND deleted_at IS NULL
		  AND status IN ('pending', 'provisioning')`, tenantID)
	if err != nil {
		return fmt.Errorf("could not move tenant to 'failed': %w", err)
	}

	return nil
}
